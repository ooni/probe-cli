package quicdialer

import (
	"context"
	"crypto/tls"

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
		return nil, NewErrDial(&err)
	}
	return sess, nil
}

type ErrDial struct {
	error
}

func NewErrDial(e *error) *ErrDial {
	return &ErrDial{*e}
}

func (e *ErrDial) Unwrap() error {
	return e.error
}

type ErrWriteTo struct {
	error
}

func NewErrWriteTo(e *error) *ErrWriteTo {
	return &ErrWriteTo{*e}
}

func (e *ErrWriteTo) Unwrap() error {
	return e.error
}

type ErrReadFrom struct {
	error
}

func NewErrReadFrom(e *error) *ErrReadFrom {
	return &ErrReadFrom{*e}
}

func (e *ErrReadFrom) Unwrap() error {
	return e.error
}
