// Package quicx contains lucas-clemente/quic-go extensions.
//
// This code introduces the UDPLikeConn, whose documentation explain
// why we need to introduce this new type. We could not put this
// code inside an existing package because it's used (as of 20 Aug 2021)
// by the netxlite package as well as by the netx package.
package quicx

import (
	"net"
	"syscall"
)

// UDPLikeConn is a net.PacketConn with some extra functions
// required to convince the QUIC library (lucas-clemente/quic-go)
// to inflate the receive buffer of the connection.
//
// The QUIC library will otherwise treat this connection as
// a dumb connection, using its ReadFrom and WriteTo methods
// as opposed to more advanced methods that are available
// under Linux and FreeBSD and improve the performance.
//
// It seems fine to avoid performance optimizations, because
// they would compilcate the implementation on our side and
// our use cases (blocking and heavy throttling) do not seem
// to require such optimizations.
//
// See https://github.com/ooni/probe/issues/1754 for a more
// comprehensive discussion.
type UDPLikeConn interface {
	// An UDPLikeConn is a quic.OOBCapablePacketConn.
	net.PacketConn

	// SetReadBuffer allows to set the read buffer.
	SetReadBuffer(bytes int) error

	// SyscallConn returns a conn suitable for calling syscalls.
	SyscallConn() (syscall.RawConn, error)
}
