package netplumbing

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"

	"github.com/bassosimone/quic-go"
)

// ErrQUICDial is an error occurred when dialing a QUIC connection.
type ErrQUICDial struct {
	error
}

// Unwrap returns the underlying error.
func (err *ErrQUICDial) Unwrap() error {
	return err.error
}

// QUICDialContext dials a QUIC connection.
func (txp *Transport) QUICDialContext(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	if config := ContextConfig(ctx); config != nil && config.Proxy != nil {
		return nil, &ErrQUICDial{ErrProxyNotImplemented}
	}
	log := txp.logger(ctx)
	log.Debugf("quicDial: %s/%s...", address, network)
	sess, err := txp.quicDialContext(ctx, network, address, tlsConfig, quicConfig)
	if err != nil {
		log.Debugf("quicDial: %s/%s... %s", address, network, err)
		return nil, &ErrQUICDial{err}
	}
	log.Debugf("quicDial: %s/%s... ok", address, network)
	return sess, nil
}

// ErrAllHandshakesFailed indicates that all QUIC handshakes failed.
type ErrAllHandshakesFailed struct {
	// Errors contains all the errors that occurred.
	Errors []error
}

// Error implements error.Error.
func (err *ErrAllHandshakesFailed) Error() string {
	return fmt.Sprintf("one or more quic handshakes failed: %#v", err.Errors)
}

// quicDialContext implements quicDialContext.
func (txp *Transport) quicDialContext(
	ctx context.Context, network, address string, tlsConfig *tls.Config,
	quicConfig *quic.Config) (quic.EarlySession, error) {
	hostname, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, err
	}
	ipaddrs, err := txp.LookupHost(ctx, hostname)
	if err != nil {
		return nil, err
	}
	aggregate := &ErrAllHandshakesFailed{}
	for _, ipaddr := range ipaddrs {
		sess, err := txp.quicHandshake(ctx, network, ipaddr, port, hostname,
			tlsConfig, quicConfig)
		if err == nil {
			return sess, nil
		}
		aggregate.Errors = append(aggregate.Errors, err)
	}
	return nil, aggregate
}
