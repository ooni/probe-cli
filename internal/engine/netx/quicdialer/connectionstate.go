package quicdialer

import (
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// connectionState returns the connectionState of a QUIC Session.
func connectionState(sess quic.EarlySession) tls.ConnectionState {
	return sess.ConnectionState().TLS.ConnectionState
}
