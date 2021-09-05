// Package quicx contains lucas-clemente/quic-go extensions.
//
// This code introduces the UDPLikeConn, whose documentation explain
// why we need to introduce this new type. We could not put this
// code inside an existing package because it's used (as of 20 Aug 2021)
// by the netxlite package as well as by the mocks package.
package quicx

import (
	"net"
	"syscall"
)

// UDPLikeConn is a net.PacketConn with some extra functions
// required to convince the QUIC library (lucas-clemente/quic-go)
// to inflate the receive buffer of the connection.
//
// The QUIC library will treat this connection as a "dumb"
// net.PacketConn, calling its ReadFrom and WriteTo methods
// as opposed to more advanced methods that are available
// under Linux and FreeBSD and improve the performance.
//
// It seems fine to avoid performance optimizations, because
// they would complicate the implementation on our side and
// our use cases (blocking and heavy throttling) do not seem
// to require such optimizations.
//
// See https://github.com/ooni/probe/issues/1754 for a more
// comprehensive discussion of UDPLikeConn.
type UDPLikeConn interface {
	// An UDPLikeConn is a net.PacketConn conn.
	net.PacketConn

	// SetReadBuffer allows setting the read buffer.
	SetReadBuffer(bytes int) error

	// SyscallConn returns a conn suitable for calling syscalls,
	// which is also instrumental to setting the read buffer.
	SyscallConn() (syscall.RawConn, error)
}
