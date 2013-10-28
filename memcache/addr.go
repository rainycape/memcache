package memcache

import (
	"net"
)

type Addr struct {
	net.Addr
	s string
}

func (a *Addr) String() string {
	return a.s
}

func NewAddr(addr net.Addr) *Addr {
	return &Addr{
		Addr: addr,
		s:    addr.String(),
	}
}
