package quicdialer

import (
	"context"
	"crypto/tls"
	"errors"

	"github.com/lucas-clemente/quic-go"
)

// ErrorWrapperDialer is a dialer that performs quic err wrapping
type ErrorWrapperDialer struct {
	Dialer ContextDialer
}

// DialContext implements ContextDialer.DialContext
func (d ErrorWrapperDialer) DialContext(
	ctx context.Context, network string, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	sess, err := d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	if err != nil {
		return nil, &ErrDial{err}
	}
	return sess, nil
}

type ErrDial struct {
	error
}

func (e *ErrDial) Unwrap() error {
	return e.error
}

type ErrWriteTo struct {
	error
}

func (e *ErrWriteTo) Unwrap() error {
	return e.error
}

type ErrReadFrom struct {
	error
}

func (e *ErrReadFrom) Unwrap() error {
	return e.error
}

// export for for testing purposes
var MockErrDial *ErrDial = &ErrDial{errors.New("mock error")}
var MockErrReadFrom *ErrReadFrom = &ErrReadFrom{errors.New("mock error")}
var MockErrWriteTo *ErrWriteTo = &ErrWriteTo{errors.New("mock error")}
