// Package quicx contains lucas-clemente/quic-go extensions.
//
// This code introduces the UDPLikeConn, whose documentation explain
// why we need to introduce this new type. We could not put this
// code inside an existing package because it's used (as of 20 Aug 2021)
// by the netxlite package as well as by the netx package.
package quicx

import (
	"net"

	"github.com/lucas-clemente/quic-go"
)

// UDPLikeConn is a quic.OOBCapablePacketConn that at the same time
// also implements the net.Conn interface. We need to implement both
// interfaces because of the panic occurred in the following PR:
//
// https://github.com/ooni/probe-cli/pull/441#issuecomment-902815439
//
// Basically, endpoint.go's NewPacketConn unconditionally casts the
// connect to a net.Conn. Thus, to avoid a panic, we need to have
// since the beginning a connection compatible with udp.Conn. Another
// strategy is to cast to after the ListenUDP, but it seems cleaner
// to avoid casting, even though it requires more code.
type UDPLikeConn interface {
	// An UDPLikeConn is a quic.OOBCapablePacketConn.
	quic.OOBCapablePacketConn

	// An UDPLikeConn is also a net.Conn.
	net.Conn
}
