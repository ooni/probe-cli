package quicdialer

import (
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// connectionState returns the ConnectionState of a QUIC Session.
func connectionState(sess quic.EarlyConnection) tls.ConnectionState {
	return sess.ConnectionState().TLS.ConnectionState
}
