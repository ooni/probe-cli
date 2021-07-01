// Package quicx contains definitions useful to implement QUIC.
package quicx

import "net"

// UDPConn is an UDP connection used by quic.
type UDPConn interface {
	// PacketConn is the underlying base interface.
	net.PacketConn

	// ReadMsgUDP behaves like net.UDPConn.ReadMsgUDP.
	ReadMsgUDP(b, oob []byte) (n, oobn, flags int, addr *net.UDPAddr, err error)
}
