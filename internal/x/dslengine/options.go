package dslengine

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type optionsValues struct {
	// activeConns controls the maximum number of active conns
	activeConns int

	// activeDNS controls the maximum number of active DNS lookups
	activeDNS int

	// netx is the underlying measuring network
	netx model.MeasuringNetwork
}

func newOptionValues(options ...Option) *optionsValues {
	values := &optionsValues{
		activeConns: 1,
		activeDNS:   1,
		netx:        &netxlite.Netx{Underlying: nil}, // implies using the host's network
	}
	for _, option := range options {
		option(values)
	}
	return values
}

// Option is an option for configuring a runtime.
type Option func(opts *optionsValues)

// OptionMeasuringNetwork configures the [model.MeasuringNetwork] to use.
func OptionMeasuringNetwork(netx model.MeasuringNetwork) Option {
	return func(opts *optionsValues) {
		opts.netx = netx
	}
}

// OptionMaxActiveConns configures the maximum number of endpoint
// measurements that we may run in parallel. If the provided value
// is <= 1, we set a maximum of 1 measurements in parallel.
func OptionMaxActiveConns(count int) Option {
	return func(opts *optionsValues) {
		switch {
		case count > 1:
			opts.activeConns = count
		default:
			opts.activeConns = 1
		}
	}
}

// OptionMaxActiveDNSLookups configures the maximum number of DNS lookup
// measurements that we may run in parallel. If the provided value
// is <= 1, we set a maximum of 1 measurements in parallel.
func OptionMaxActiveDNSLookups(count int) Option {
	return func(opts *optionsValues) {
		switch {
		case count > 1:
			opts.activeDNS = count
		default:
			opts.activeDNS = 1
		}
	}
}
