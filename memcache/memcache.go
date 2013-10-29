/*
Copyright 2011 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package memcache provides a client for the memcached cache server.
package memcache

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"runtime"
	"sync"
	"time"
)

// Similar to:
// http://code.google.com/appengine/docs/go/memcache/reference.html

var (
	// ErrCacheMiss means that a Get failed because the item wasn't present.
	ErrCacheMiss = errors.New("memcache: cache miss")

	// ErrCASConflict means that a CompareAndSwap call failed due to the
	// cached value being modified between the Get and the CompareAndSwap.
	// If the cached value was simply evicted rather than replaced,
	// ErrNotStored will be returned instead.
	ErrCASConflict = errors.New("memcache: compare-and-swap conflict")

	// ErrNotStored means that a conditional write operation (i.e. Add or
	// CompareAndSwap) failed because the condition was not satisfied.
	ErrNotStored = errors.New("memcache: item not stored")

	// ErrServer means that a server error occurred.
	ErrServerError = errors.New("memcache: server error")

	// ErrNoStats means that no statistics were available.
	ErrNoStats = errors.New("memcache: no statistics available")

	// ErrMalformedKey is returned when an invalid key is used.
	// Keys must be at maximum 250 bytes long, ASCII, and not
	// contain whitespace or control characters.
	ErrMalformedKey = errors.New("malformed: key is too long or contains invalid characters")

	// ErrNoServers is returned when no servers are configured or available.
	ErrNoServers = errors.New("memcache: no servers configured or available")

	// ErrBadMagic is returned when the magic number in a response is not valid.
	ErrBadMagic = errors.New("memcache: bad magic number in response")

	// ErrBadIncrDec is returned when performing a incr/decr on non-numeric values.
	ErrBadIncrDec = errors.New("memcache: incr or decr on non-numeric value")
)

// DefaultTimeout is the default socket read/write timeout.
const DefaultTimeout = time.Duration(100) * time.Millisecond

const (
	buffered            = 8 // arbitrary buffered channel size, for readability
	maxIdleConnsPerAddr = 2 // TODO(bradfitz): make this configurable?
)

var (
	zero8  = []byte{0}
	zero16 = []byte{0, 0}
	zero32 = []byte{0, 0, 0, 0}
	zero64 = []byte{0, 0, 0, 0, 0, 0, 0, 0}
)

type command uint8

const (
	cmdGet = iota
	cmdSet
	cmdAdd
	cmdReplace
	cmdDelete
	cmdIncr
	cmdDecr
	cmdQuit
	cmdFlush
	cmdGetQ
	cmdNoop
	cmdVersion
	cmdGetK
	cmdGetKQ
)

type response uint16

const (
	respOk = iota
	respKeyNotFound
	respKeyExists
	respValueTooLarge
	respInvalidArgs
	respItemNotStored
	respInvalidIncrDecr
	respWrongVBucket
	respAuthErr
	respAuthContinue
	respUnknownCmd   = 0x81
	respOOM          = 0x82
	respNotSupported = 0x83
	respInternalErr  = 0x85
	respBusy         = 0x85
	respTemporaryErr = 0x86
)

func (r response) asError() error {
	switch r {
	case respKeyNotFound:
		return ErrCacheMiss
	case respKeyExists:
		return ErrNotStored
	case respInvalidIncrDecr:
		return ErrBadIncrDec
	case respItemNotStored:
		return ErrNotStored
	}
	return r
}

func (r response) Error() string {
	switch r {
	case respOk:
		return "Ok"
	case respKeyNotFound:
		return "key not found"
	case respKeyExists:
		return "key already exists"
	case respValueTooLarge:
		return "value too large"
	case respInvalidArgs:
		return "invalid arguments"
	case respItemNotStored:
		return "item not stored"
	case respInvalidIncrDecr:
		return "incr/decr on non-numeric value"
	case respWrongVBucket:
		return "wrong vbucket"
	case respAuthErr:
		return "auth error"
	case respAuthContinue:
		return "auth continue"
	}
	return ""
}

const (
	reqMagic  uint8 = 0x80
	respMagic uint8 = 0x81
)

// resumableError returns true if err is only a protocol-level cache error.
// This is used to determine whether or not a server connection should
// be re-used or not. If an error occurs, by default we don't reuse the
// connection, unless it was just a cache error.
func resumableError(err error) bool {
	switch err {
	case ErrCacheMiss, ErrCASConflict, ErrNotStored, ErrMalformedKey, ErrBadIncrDec:
		return true
	}
	return false
}

func legalKey(key string) bool {
	if len(key) > 250 {
		return false
	}
	for i := 0; i < len(key); i++ {
		if key[i] <= ' ' || key[i] > 0x7e {
			return false
		}
	}
	return true
}

func poolSize() int {
	s := 8
	if mp := runtime.GOMAXPROCS(0); mp > s {
		s = mp
	}
	return s
}

// New returns a memcache client using the provided server(s)
// with equal weight. If a server is listed multiple times,
// it gets a proportional amount of weight.
func New(server ...string) *Client {
	ss := new(ServerList)
	ss.SetServers(server...)
	return NewFromSelector(ss)
}

// NewFromSelector returns a new Client using the provided ServerSelector.
func NewFromSelector(ss ServerSelector) *Client {
	return &Client{
		timeout:  DefaultTimeout,
		selector: ss,
		freeconn: make(map[string]chan *conn),
		bufPool:  make(chan *bytes.Buffer, poolSize()),
	}
}

// Client is a memcache client.
// It is safe for unlocked use by multiple concurrent goroutines.
type Client struct {
	timeout  time.Duration
	selector ServerSelector
	mu       sync.RWMutex
	freeconn map[string]chan *conn
	bufPool  chan *bytes.Buffer
}

// Timeout returns the socket read/write timeout. By default, it's
// DefaultTimeout.
func (c *Client) Timeout() time.Duration {
	return c.timeout
}

// SetTimeout specifies the socket read/write timeout.
// If zero, DefaultTimeout is used.
func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout == time.Duration(0) {
		timeout = DefaultTimeout
	}
	c.timeout = timeout
}

// Item is an item to be got or stored in a memcached server.
type Item struct {
	// Key is the Item's key (250 bytes maximum).
	Key string

	// Value is the Item's value.
	Value []byte

	// Object is the Item's value for use with a Codec.
	Object interface{}

	// Flags are server-opaque flags whose semantics are entirely
	// up to the app.
	Flags uint32

	// Expiration is the cache expiration time, in seconds: either a relative
	// time from now (up to 1 month), or an absolute Unix epoch time.
	// Zero means the Item has no expiration time.
	Expiration int32

	// Compare and swap ID.
	casid uint64
}

// conn is a connection to a server.
type conn struct {
	nc   net.Conn
	addr net.Addr
	c    *Client
}

// release returns this connection back to the client's free pool
func (cn *conn) release() {
	cn.c.putFreeConn(cn.addr, cn)
}

func (cn *conn) extendDeadline() {
	cn.nc.SetDeadline(time.Now().Add(cn.c.timeout))
}

// condRelease releases this connection if the error pointed to by err
// is is nil (not an error) or is only a protocol level error (e.g. a
// cache miss).  The purpose is to not recycle TCP connections that
// are bad.
func (cn *conn) condRelease(err *error) {
	if *err == nil || resumableError(*err) {
		cn.release()
	} else {
		cn.nc.Close()
	}
}

func (cn *conn) condClose(err *error) {
	if *err != nil {
		cn.nc.Close()
	}
}

func (c *Client) putFreeConn(addr net.Addr, cn *conn) {
	c.mu.RLock()
	freelist := c.freeconn[addr.String()]
	c.mu.RUnlock()
	if freelist == nil {
		freelist = make(chan *conn, maxIdleConnsPerAddr)
		c.mu.Lock()
		c.freeconn[addr.String()] = freelist
		c.mu.Unlock()
	}
	select {
	case freelist <- cn:
		break
	default:
		cn.nc.Close()
	}
}

func (c *Client) getFreeConn(addr net.Addr) (cn *conn, ok bool) {
	c.mu.RLock()
	freelist := c.freeconn[addr.String()]
	c.mu.RUnlock()
	if freelist == nil {
		return nil, false
	}
	select {
	case cn := <-freelist:
		return cn, true
	default:
		return nil, false
	}
}

// ConnectTimeoutError is the error type used when it takes
// too long to connect to the desired host. This level of
// detail can generally be ignored.
type ConnectTimeoutError struct {
	Addr net.Addr
}

func (cte *ConnectTimeoutError) Error() string {
	return "memcache: connect timeout to " + cte.Addr.String()
}

func (c *Client) dial(addr *Addr) (net.Conn, error) {
	type connError struct {
		cn  net.Conn
		err error
	}
	ch := make(chan connError)
	go func() {
		nc, err := net.Dial(addr.Network(), addr.String())
		ch <- connError{nc, err}
	}()
	select {
	case ce := <-ch:
		return ce.cn, ce.err
	case <-time.After(c.timeout):
		// Too slow. Fall through.
	}
	// Close the conn if it does end up finally coming in
	go func() {
		ce := <-ch
		if ce.err == nil {
			ce.cn.Close()
		}
	}()
	return nil, &ConnectTimeoutError{addr}
}

func (c *Client) getConn(addr *Addr) (*conn, error) {
	cn, ok := c.getFreeConn(addr)
	if !ok {
		nc, err := c.dial(addr)
		if err != nil {
			return nil, err
		}
		cn = &conn{
			nc:   nc,
			addr: addr,
			c:    c,
		}
	}
	cn.extendDeadline()
	return cn, nil
}

// Get gets the item for the given key. ErrCacheMiss is returned for a
// memcache cache miss. The key must be at most 250 bytes in length.
func (c *Client) Get(key string) (*Item, error) {
	cn, err := c.sendCommand(key, cmdGet, nil, nil)
	if err != nil {
		return nil, err
	}
	return c.parseItemResponse(key, cn, true)
}

func (c *Client) sendCommand(key string, cmd command, item *Item, extras []byte) (*conn, error) {
	if !legalKey(key) {
		return nil, ErrMalformedKey
	}
	addr, err := c.selector.PickServer(key)
	if err != nil {
		return nil, err
	}
	cn, err := c.getConn(addr)
	if err != nil {
		return nil, err
	}
	defer cn.condClose(&err)
	err = c.sendConnCommand(cn, key, cmd, item, extras)
	return cn, err
}

func (c *Client) sendConnCommand(cn *conn, key string, cmd command, item *Item, extras []byte) (err error) {
	var buf *bytes.Buffer
	select {
	case buf = <-c.bufPool:
		buf.Reset()
	default:
		buf = bytes.NewBuffer(nil)
		// 24 is header size
		buf.Grow(24)
	}
	buf.WriteByte(reqMagic)
	buf.WriteByte(byte(cmd))
	kl := len(key)
	el := len(extras)
	// Key length
	binary.Write(buf, binary.BigEndian, uint16(kl))
	// Extras length
	buf.WriteByte(byte(el))
	// Data type
	buf.WriteByte(0)
	// VBucket
	buf.Write(zero16)
	// Total body length
	bl := uint32(kl + el)
	if item != nil {
		bl += uint32(len(item.Value))
	}
	binary.Write(buf, binary.BigEndian, bl)
	// Opaque
	buf.Write(zero32)
	// CAS
	if item != nil && item.casid != 0 {
		binary.Write(buf, binary.BigEndian, item.casid)
	} else {
		buf.Write(zero64)
	}
	// Extras
	if el > 0 {
		buf.Write(extras)
	}
	if kl > 0 {
		// Key itself
		buf.Write(stobs(key))
	}
	if _, err = cn.nc.Write(buf.Bytes()); err != nil {
		return err
	}
	select {
	case c.bufPool <- buf:
	default:
	}
	if item != nil {
		if _, err = cn.nc.Write(item.Value); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) parseResponse(cn *conn) ([]byte, []byte, []byte, []byte, error) {
	var err error
	hdr := make([]byte, 24)
	if _, err = cn.nc.Read(hdr); err != nil {
		return nil, nil, nil, nil, err
	}
	if hdr[0] != respMagic {
		return nil, nil, nil, nil, ErrBadMagic
	}
	total := int(binary.BigEndian.Uint32(hdr[8:12]))
	status := binary.BigEndian.Uint16(hdr[6:8])
	if status != respOk {
		if _, err = io.CopyN(ioutil.Discard, cn.nc, int64(total)); err != nil {
			return nil, nil, nil, nil, err
		}
		return nil, nil, nil, nil, response(status).asError()
	}
	var extras []byte
	el := int(hdr[4])
	if el > 0 {
		extras = make([]byte, el)
		if _, err = cn.nc.Read(extras); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	var key []byte
	kl := int(binary.BigEndian.Uint16(hdr[2:4]))
	if kl > 0 {
		key = make([]byte, int(kl))
		if _, err = cn.nc.Read(key); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	var body []byte
	bl := total - el - kl
	if bl > 0 {
		body = make([]byte, bl)
		if _, err = cn.nc.Read(body); err != nil {
			return nil, nil, nil, nil, err
		}
	}
	return hdr, key, extras, body, nil
}

func (c *Client) parseUintResponse(cn *conn) (uint64, error) {
	_, _, _, body, err := c.parseResponse(cn)
	cn.condRelease(&err)
	if err != nil {
		return 0, err
	}
	return binary.BigEndian.Uint64(body), nil
}

func (c *Client) parseItemResponse(key string, cn *conn, release bool) (*Item, error) {
	hdr, k, extras, body, err := c.parseResponse(cn)
	if release {
		cn.condRelease(&err)
	}
	if err != nil {
		return nil, err
	}
	var flags uint32
	if len(extras) > 0 {
		flags = binary.BigEndian.Uint32(extras)
	}
	if key == "" && len(k) > 0 {
		key = string(k)
	}
	return &Item{
		Key:   key,
		Value: body,
		Flags: flags,
		casid: binary.BigEndian.Uint64(hdr[16:24]),
	}, nil
}

// GetMulti is a batch version of Get. The returned map from keys to
// items may have fewer elements than the input slice, due to memcache
// cache misses. Each key must be at most 250 bytes in length.
// If no error is returned, the returned map will also be non-nil.
func (c *Client) GetMulti(keys []string) (map[string]*Item, error) {
	keyMap := make(map[*Addr][]string)
	for _, key := range keys {
		if !legalKey(key) {
			return nil, ErrMalformedKey
		}
		addr, err := c.selector.PickServer(key)
		if err != nil {
			return nil, err
		}
		keyMap[addr] = append(keyMap[addr], key)
	}

	var chs []chan *Item
	for addr, keys := range keyMap {
		ch := make(chan *Item)
		chs = append(chs, ch)
		go func(addr *Addr, keys []string, ch chan *Item) {
			defer close(ch)
			cn, err := c.getConn(addr)
			if err != nil {
				return
			}
			defer cn.condRelease(&err)
			for _, k := range keys {
				if err = c.sendConnCommand(cn, k, cmdGetKQ, nil, nil); err != nil {
					return
				}
			}
			if err = c.sendConnCommand(cn, "", cmdNoop, nil, nil); err != nil {
				return
			}
			var item *Item
			for {
				item, err = c.parseItemResponse("", cn, false)
				if item == nil || item.Key == "" {
					// Noop response
					break
				}
				ch <- item
			}
		}(addr, keys, ch)
	}

	m := make(map[string]*Item)
	for _, ch := range chs {
		for item := range ch {
			m[item.Key] = item
		}
	}
	return m, nil
}

// Set writes the given item, unconditionally.
func (c *Client) Set(item *Item) error {
	return c.populateOne(cmdSet, item, false)
}

// Add writes the given item, if no value already exists for its
// key. ErrNotStored is returned if that condition is not met.
func (c *Client) Add(item *Item) error {
	return c.populateOne(cmdAdd, item, false)
}

// CompareAndSwap writes the given item that was previously returned
// by Get, if the value was neither modified or evicted between the
// Get and the CompareAndSwap calls. The item's Key should not change
// between calls but all other item fields may differ. ErrCASConflict
// is returned if the value was modified in between the
// calls. ErrNotStored is returned if the value was evicted in between
// the calls.
func (c *Client) CompareAndSwap(item *Item) error {
	return c.populateOne(cmdSet, item, true)
}

func (c *Client) cas(nc net.Conn, item *Item) error {
	return c.populateOne(cmdSet, item, true)
}

func (c *Client) populateOne(cmd command, item *Item, cas bool) error {
	if item == nil || !legalKey(item.Key) {
		return ErrMalformedKey
	}
	extras := make([]byte, 8)
	binary.BigEndian.PutUint32(extras, item.Flags)
	binary.BigEndian.PutUint32(extras[4:8], uint32(item.Expiration))
	if !cas && item.casid != 0 {
		item.casid = 0
	}
	cn, err := c.sendCommand(item.Key, cmd, item, extras)
	if err != nil {
		return err
	}
	hdr, _, _, _, err := c.parseResponse(cn)
	cn.condRelease(&err)
	if err != nil {
		return err
	}
	item.casid = binary.BigEndian.Uint64(hdr[16:24])
	return nil
}

// Delete deletes the item with the provided key. The error ErrCacheMiss is
// returned if the item didn't already exist in the cache.
func (c *Client) Delete(key string) error {
	cn, err := c.sendCommand(key, cmdDelete, nil, nil)
	if err != nil {
		return err
	}
	_, _, _, _, err = c.parseResponse(cn)
	cn.condRelease(&err)
	return err
}

// Increment atomically increments key by delta. The return value is
// the new value after being incremented or an error. If the value
// didn't exist in memcached the error is ErrCacheMiss. The value in
// memcached must be an decimal number, or an error will be returned.
// On 64-bit overflow, the new value wraps around.
func (c *Client) Increment(key string, delta uint64) (newValue uint64, err error) {
	return c.incrDecr(cmdIncr, key, delta)
}

// Decrement atomically decrements key by delta. The return value is
// the new value after being decremented or an error. If the value
// didn't exist in memcached the error is ErrCacheMiss. The value in
// memcached must be an decimal number, or an error will be returned.
// On underflow, the new value is capped at zero and does not wrap
// around.
func (c *Client) Decrement(key string, delta uint64) (newValue uint64, err error) {
	return c.incrDecr(cmdDecr, key, delta)
}

func (c *Client) incrDecr(cmd command, key string, delta uint64) (uint64, error) {
	extras := make([]byte, 20)
	binary.BigEndian.PutUint64(extras, delta)
	// Set expiration to 0xfffffff, so the command fails if the key
	// does not exist.
	for ii := 16; ii < 20; ii++ {
		extras[ii] = 0xff
	}
	cn, err := c.sendCommand(key, cmd, nil, extras)
	if err != nil {
		return 0, err
	}
	return c.parseUintResponse(cn)
}
