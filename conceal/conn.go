package conceal

import (
	"net"
	"syscall"

	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type UDPConn interface {
	net.PacketConn
	SyscallConn() (syscall.RawConn, error)
	ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
	WriteMsgUDP(b, oob []byte, addr *net.UDPAddr) (n, oobn int, err error)
}

var _ ipv4.Message = ipv6.Message{}

type BatchConn interface {
	ReadBatch(ms []ipv4.Message, flags int) (n int, err error)
	WriteBatch(ms []ipv4.Message, flags int) (n int, err error)
}
