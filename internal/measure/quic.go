package measure

import (
	"context"
	"crypto/tls"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// QUICHandshaker creates QUIC sessions.
type QUICHandshaker interface {
	QUICHandshake(ctx context.Context, address string,
		tlsConfig *tls.Config) *QUICHandshakeResult
}

// QUICHandshakeResult is the result of QUICHandshake.
type QUICHandshakeResult struct {
	// Address is the endpoint address we want to use (e.g., "1.1.1.1:443")
	Address string `json:"address"`

	// Config contains the TLS config.
	Config *TLSConfig `json:"config"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Failure contains the error (nil on success).
	Failure error `json:"failure"`

	// ConnectionState contains the connection state (only set on success).
	ConnectionState *TLSConnectionState `json:"connection_state"`

	// Sess is the QUIC session (only set on success).
	Sess quic.EarlySession `json:"-"`
}

// NewQUICHandshaker creates a new QUICHandshaker instance.
func NewQUICHandshaker(begin time.Time, logger Logger, trace *Trace) QUICHandshaker {
	return &quicHandshaker{begin: begin, logger: logger, trace: trace}
}

type quicHandshaker struct {
	begin  time.Time
	logger Logger
	trace  *Trace
}

func (qh *quicHandshaker) QUICHandshake(ctx context.Context, address string,
	tlsConfig *tls.Config) *QUICHandshakeResult {
	m := &QUICHandshakeResult{
		Address: address,
		Config:  newTLSConfig(tlsConfig),
		Started: time.Since(qh.begin),
	}
	handshaker := netxlite.NewQUICDialerWithoutResolver(
		qh.trace.wrapQUICListener(netxlite.NewQUICListener()), qh.logger)
	defer handshaker.CloseIdleConnections() // respect the protocol
	sess, err := handshaker.DialContext(
		ctx, "udp", address, tlsConfig, &quic.Config{})
	m.Completed = time.Since(qh.begin)
	if err != nil {
		m.Failure = err
		return m
	}
	select {
	case <-sess.HandshakeComplete().Done():
	case <-ctx.Done():
		m.Failure = err
		return m
	}
	state := sess.ConnectionState().TLS
	m.ConnectionState = newTLSConnectionState(&tls.ConnectionState{
		Version:                     state.Version,
		HandshakeComplete:           state.HandshakeComplete,
		DidResume:                   state.DidResume,
		CipherSuite:                 state.CipherSuite,
		NegotiatedProtocol:          state.NegotiatedProtocol,
		NegotiatedProtocolIsMutual:  true, // this field is deprecated
		ServerName:                  state.ServerName,
		PeerCertificates:            state.PeerCertificates,
		VerifiedChains:              state.VerifiedChains,
		SignedCertificateTimestamps: state.SignedCertificateTimestamps,
		OCSPResponse:                state.OCSPResponse,
		TLSUnique:                   nil, // this field is deprecated
	})
	m.Sess = sess
	return m
}
