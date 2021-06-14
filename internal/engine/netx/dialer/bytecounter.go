package dialer

import (
	"context"
	"net"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
)

// byteCounterDialer is a byte-counting-aware dialer. To perform byte counting, you
// should make sure that you insert this dialer in the dialing chain.
type byteCounterDialer struct {
	Dialer
}

// DialContext implements Dialer.DialContext
func (d *byteCounterDialer) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	conn, err := d.Dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	exp := contextExperimentByteCounter(ctx)
	sess := contextSessionByteCounter(ctx)
	if exp == nil && sess == nil {
		return conn, nil // no point in wrapping
	}
	return &byteCounterConnWrapper{Conn: conn, exp: exp, sess: sess}, nil
}

type byteCounterSessionKey struct{}

// contextSessionByteCounter retrieves the session byte counter from the context
func contextSessionByteCounter(ctx context.Context) *bytecounter.Counter {
	counter, _ := ctx.Value(byteCounterSessionKey{}).(*bytecounter.Counter)
	return counter
}

// WithSessionByteCounter assigns the session byte counter to the context.
func WithSessionByteCounter(ctx context.Context, counter *bytecounter.Counter) context.Context {
	return context.WithValue(ctx, byteCounterSessionKey{}, counter)
}

type byteCounterExperimentKey struct{}

// contextExperimentByteCounter retrieves the experiment byte counter from the context
func contextExperimentByteCounter(ctx context.Context) *bytecounter.Counter {
	counter, _ := ctx.Value(byteCounterExperimentKey{}).(*bytecounter.Counter)
	return counter
}

// WithExperimentByteCounter assigns the experiment byte counter to the context.
func WithExperimentByteCounter(ctx context.Context, counter *bytecounter.Counter) context.Context {
	return context.WithValue(ctx, byteCounterExperimentKey{}, counter)
}

type byteCounterConnWrapper struct {
	net.Conn
	exp  *bytecounter.Counter
	sess *bytecounter.Counter
}

func (c *byteCounterConnWrapper) Read(p []byte) (int, error) {
	count, err := c.Conn.Read(p)
	if c.exp != nil {
		c.exp.CountBytesReceived(count)
	}
	if c.sess != nil {
		c.sess.CountBytesReceived(count)
	}
	return count, err
}

func (c *byteCounterConnWrapper) Write(p []byte) (int, error) {
	count, err := c.Conn.Write(p)
	if c.exp != nil {
		c.exp.CountBytesSent(count)
	}
	if c.sess != nil {
		c.sess.CountBytesSent(count)
	}
	return count, err
}
