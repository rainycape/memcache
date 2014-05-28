Memcache Client in Go (golang)
=============================

## Installing

    $ go get github.com/rainycape/memcache

After this command *memcache* is ready to use. Its source will be in:

    $GOPATH/src/pkg/github.com/rainycape/memcache

You can use `go get -u -a` for update all installed packages.

## Example

    import (
            "github.com/rainycape/memcache"
    )

    func main() {
         mc := memcache.New("10.0.0.1:11211", "10.0.0.2:11211", "10.0.0.3:11212")
         mc.Set(&memcache.Item{Key: "foo", Value: []byte("my value")})

         it, err := mc.Get("foo")
         ...
    }

## Documentation

## package memcache
--
    import "github.com/rainycape/memcache"

Package memcache provides a client for the memcached cache server.

## Usage

```go
const DefaultTimeout = time.Duration(100) * time.Millisecond
```
DefaultTimeout is the default socket read/write timeout.

```go
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
```

#### type Addr

```go
type Addr struct {
	net.Addr
}
```


#### func  NewAddr

```go
func NewAddr(addr net.Addr) *Addr
```

#### func (*Addr) String

```go
func (a *Addr) String() string
```

#### type Client

```go
type Client struct {
}
```

Client is a memcache client. It is safe for unlocked use by multiple concurrent
goroutines.

#### func  New

```go
func New(server ...string) *Client
```
New returns a memcache client using the provided server(s) with equal weight. If
a server is listed multiple times, it gets a proportional amount of weight.

#### func  NewFromSelector

```go
func NewFromSelector(ss ServerSelector) *Client
```
NewFromSelector returns a new Client using the provided ServerSelector.

#### func (*Client) Add

```go
func (c *Client) Add(item *Item) error
```
Add writes the given item, if no value already exists for its key. ErrNotStored
is returned if that condition is not met.

#### func (*Client) Close

```go
func (c *Client) Close() error
```
Close closes all currently open connections.

#### func (*Client) CompareAndSwap

```go
func (c *Client) CompareAndSwap(item *Item) error
```
CompareAndSwap writes the given item that was previously returned by Get, if the
value was neither modified or evicted between the Get and the CompareAndSwap
calls. The item's Key should not change between calls but all other item fields
may differ. ErrCASConflict is returned if the value was modified in between the
calls. ErrNotStored is returned if the value was evicted in between the calls.

#### func (*Client) Decrement

```go
func (c *Client) Decrement(key string, delta uint64) (newValue uint64, err error)
```
Decrement atomically decrements key by delta. The return value is the new value
after being decremented or an error. If the value didn't exist in memcached the
error is ErrCacheMiss. The value in memcached must be an decimal number, or an
error will be returned. On underflow, the new value is capped at zero and does
not wrap around.

#### func (*Client) Delete

```go
func (c *Client) Delete(key string) error
```
Delete deletes the item with the provided key. The error ErrCacheMiss is
returned if the item didn't already exist in the cache.

#### func (*Client) Get

```go
func (c *Client) Get(key string) (*Item, error)
```
Get gets the item for the given key. ErrCacheMiss is returned for a memcache
cache miss. The key must be at most 250 bytes in length.

#### func (*Client) GetMulti

```go
func (c *Client) GetMulti(keys []string) (map[string]*Item, error)
```
GetMulti is a batch version of Get. The returned map from keys to items may have
fewer elements than the input slice, due to memcache cache misses. Each key must
be at most 250 bytes in length. If no error is returned, the returned map will
also be non-nil.

#### func (*Client) Increment

```go
func (c *Client) Increment(key string, delta uint64) (newValue uint64, err error)
```
Increment atomically increments key by delta. The return value is the new value
after being incremented or an error. If the value didn't exist in memcached the
error is ErrCacheMiss. The value in memcached must be an decimal number, or an
error will be returned. On 64-bit overflow, the new value wraps around.

#### func (*Client) MaxIdleConnsPerAddr

```go
func (c *Client) MaxIdleConnsPerAddr() int
```
MaxIdleConnsPerAddr returns the maximum number of idle connections kept per
server address.

#### func (*Client) Set

```go
func (c *Client) Set(item *Item) error
```
Set writes the given item, unconditionally.

#### func (*Client) SetMaxIdleConnsPerAddr

```go
func (c *Client) SetMaxIdleConnsPerAddr(maxIdle int)
```
SetMaxIdleConnsPerAddr changes the maximum number of idle connections kept per
server. If maxIdle < 0, no idle connections are kept. If maxIdle == 0, the
default number (currently 2) is used.

#### func (*Client) SetTimeout

```go
func (c *Client) SetTimeout(timeout time.Duration)
```
SetTimeout specifies the socket read/write timeout. If zero, DefaultTimeout is
used. If < 0, there's no timeout. This method must be called before any
connections to the memcached server are opened.

#### func (*Client) Timeout

```go
func (c *Client) Timeout() time.Duration
```
Timeout returns the socket read/write timeout. By default, it's DefaultTimeout.

#### type ConnectTimeoutError

```go
type ConnectTimeoutError struct {
	Addr net.Addr
}
```

ConnectTimeoutError is the error type used when it takes too long to connect to
the desired host. This level of detail can generally be ignored.

#### func (*ConnectTimeoutError) Error

```go
func (cte *ConnectTimeoutError) Error() string
```

#### func (*ConnectTimeoutError) Temporary

```go
func (cte *ConnectTimeoutError) Temporary() bool
```

#### func (*ConnectTimeoutError) Timeout

```go
func (cte *ConnectTimeoutError) Timeout() bool
```

#### type Item

```go
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
}
```

Item is an item to be got or stored in a memcached server.

#### type ServerList

```go
type ServerList struct {
}
```

ServerList is a simple ServerSelector. Its zero value is usable.

#### func (*ServerList) PickServer

```go
func (ss *ServerList) PickServer(key string) (*Addr, error)
```

#### func (*ServerList) SetServers

```go
func (ss *ServerList) SetServers(servers ...string) error
```
SetServers changes a ServerList's set of servers at runtime and is threadsafe.

Each server is given equal weight. A server is given more weight if it's listed
multiple times.

SetServers returns an error if any of the server names fail to resolve. No
attempt is made to connect to the server. If any error is returned, no changes
are made to the ServerList.

#### type ServerSelector

```go
type ServerSelector interface {
	// PickServer returns the server address that a given item
	// should be shared onto.
	PickServer(key string) (*Addr, error)
}
```

ServerSelector is the interface that selects a memcache server as a function of
the item's key.

All ServerSelector implementations must be threadsafe.

## About

This is a memcache client library for the Go programming language
(http://golang.org/). This is a high performance fork of the original
library at http://github.com/bradfitz/gomemcache. The following is
a comparison between the original library and this one:

    benchmark                               old ns/op    new ns/op    delta
    BenchmarkSetGet                            214443       138200  -35.55%
    BenchmarkSetGetLarge                       262164       146594  -44.08%
    BenchmarkConcurrentSetGetSmall10_100     82561221     51282962  -37.88%
    BenchmarkConcurrentSetGetLarge10_100     96067285     63887183  -33.50%
    BenchmarkConcurrentSetGetSmall20_100    152834658     75607154  -50.53%
    BenchmarkConcurrentSetGetLarge20_100    202574186     96010615  -52.60%

    benchmark                                old MB/s     new MB/s  speedup
    BenchmarkSetGet                              0.03         0.04    1.33x
    BenchmarkSetGetLarge                         4.82         8.62    1.79x
    BenchmarkConcurrentSetGetSmall10_100         0.07         0.12    1.71x
    BenchmarkConcurrentSetGetLarge10_100        13.16        19.78    1.50x
    BenchmarkConcurrentSetGetSmall20_100         0.08         0.16    2.00x
    BenchmarkConcurrentSetGetLarge20_100        12.48        26.33    2.11x

    benchmark                              old allocs   new allocs    delta
    BenchmarkSetGet                                18            6  -66.67%
    BenchmarkSetGetLarge                           19            6  -68.42%
    BenchmarkConcurrentSetGetSmall10_100        58469         6199  -89.40%
    BenchmarkConcurrentSetGetLarge10_100        59848         6196  -89.65%
    BenchmarkConcurrentSetGetSmall20_100       117177        12432  -89.39%
    BenchmarkConcurrentSetGetLarge20_100       120173        12413  -89.67%

    benchmark                               old bytes    new bytes    delta
    BenchmarkSetGet                              2479          170  -93.14%
    BenchmarkSetGetLarge                         7537         1184  -84.29%
    BenchmarkConcurrentSetGetSmall10_100      3101520       187245  -93.96%
    BenchmarkConcurrentSetGetLarge10_100      8330341      1197783  -85.62%
    BenchmarkConcurrentSetGetSmall20_100      6318072       374977  -94.07%
    BenchmarkConcurrentSetGetLarge20_100     16884200      2398491  -85.79%
