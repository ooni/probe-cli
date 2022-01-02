package model

import (
	"net"
	"syscall"
)

//
// Network extensions
//

// HTTPSSvc is the reply to an HTTPS DNS query.
type HTTPSSvc struct {
	// ALPN contains the ALPNs inside the HTTPS reply.
	ALPN []string

	// IPv4 contains the IPv4 hints (which may be empty).
	IPv4 []string

	// IPv6 contains the IPv6 hints (which may be empty).
	IPv6 []string
}

// UDPLikeConn is a net.PacketConn with some extra functions
// required to convince the QUIC library (lucas-clemente/quic-go)
// to inflate the receive buffer of the connection.
//
// The QUIC library will treat this connection as a "dumb"
// net.PacketConn, calling its ReadFrom and WriteTo methods
// as opposed to more efficient methods that are available
// under Linux and (maybe?) FreeBSD.
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
