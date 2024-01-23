package torx

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

// PTInfo provides info about a running pluggable transport.
type PTInfo interface {
	// AsClientTransportPluginArgument returns the string to pass to
	// the tor ClientTransportPlugin command line option.
	AsClientTransportPluginArgument() string

	// AsBridgeArgument returns the string to pass to
	// the tor Bridge command line option.
	AsBridgeArgument() string
}

// PTService is a service running a pluggable transport.
type PTService interface {
	// PTInfo provides tor with information about the pluggable transport.
	PTInfo

	// Stop shuts down the PTService.
	Stop()
}

// PTServiceSnowflakeOption is an option for configuring [NewPTServiceSnowflake].
type PTServiceSnowflakeOption func(config *ptServiceSnowflakeOptions)

type ptServiceSnowflakeOptions struct {
	rendezvousMethod string
}

func newPTServiceSnowflakeOptions() *ptServiceSnowflakeOptions {
	return &ptServiceSnowflakeOptions{
		rendezvousMethod: ptx.NewSnowflakeRendezvousMethodDomainFronting().Name(),
	}
}

// PTServiceSnowflakeOptionRendezvousMethod sets the rendezvous method.
func PTServiceSnowflakeOptionRendezvousMethod(method string) PTServiceSnowflakeOption {
	return func(config *ptServiceSnowflakeOptions) {
		config.rendezvousMethod = method
	}
}

type ptServiceSnowflake struct {
	dialer   *ptx.SnowflakeDialer
	listener *ptx.Listener
}

var _ PTService = &ptServiceSnowflake{}

// NewPTServiceSnowflake starts snowflake as a service and
// returns the corresponding service descriptor.
func NewPTServiceSnowflake(logger model.Logger, options ...PTServiceSnowflakeOption) (PTService, error) {
	// honour options
	config := newPTServiceSnowflakeOptions()
	for _, option := range options {
		option(config)
	}

	// create dialer
	method, err := ptx.NewSnowflakeRendezvousMethod(config.rendezvousMethod)
	if err != nil {
		return nil, err
	}
	dialer := ptx.NewSnowflakeDialerWithRendezvousMethod(method)

	// TODO(bassosimone): we should provide proper byte counters here

	// create PT listener
	listener := &ptx.Listener{
		ExperimentByteCounter: nil,
		ListenSocks:           nil,
		Logger:                logger,
		PTDialer:              dialer,
		SessionByteCounter:    nil,
	}
	if err := listener.Start(); err != nil {
		return nil, err
	}

	// return the service
	svc := &ptServiceSnowflake{
		dialer:   dialer,
		listener: listener,
	}
	return svc, nil
}

// AsBridgeArgument implements PTService.
func (pts *ptServiceSnowflake) AsBridgeArgument() string {
	return pts.dialer.AsBridgeArgument()
}

// AsClientTransportPluginArgument implements PTService.
func (pts *ptServiceSnowflake) AsClientTransportPluginArgument() string {
	return pts.listener.AsClientTransportPluginArgument()
}

// Stop implements PTService.
func (pts *ptServiceSnowflake) Stop() {
	pts.listener.Stop()
}
