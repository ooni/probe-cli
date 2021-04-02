package quicdialer

import (
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
)

// ConnectionState returns the ConnectionState of a QUIC Session.
func ConnectionState(sess quic.EarlySession) tls.ConnectionState {
	return sess.ConnectionState().TLS.ConnectionState
}
