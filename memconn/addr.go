package memconn

import "net"

type addr struct{}

var _ net.Addr = (*addr)(nil)

func (a addr) Network() string { return "memconn" }
func (a addr) String() string  { return "memconn" }
