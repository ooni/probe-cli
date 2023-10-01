package dslmodel

import "github.com/ooni/probe-cli/v3/internal/model"

// Trace traces execution and (typically) collects OONI observations.
type Trace interface {
	// NewDialerWithoutResolver returns a dialer that saves observations into this trace.
	//
	// Caveat: the dialer wrappers are there to implement the model.MeasuringNetwork
	// interface, but they're not used by this function.
	NewDialerWithoutResolver(dl model.DebugLogger, wrappers ...model.DialerWrapper) model.Dialer

	// NewStdlibResolver returns a resolver that saves observations into this trace.
	NewStdlibResolver(logger model.DebugLogger) model.Resolver

	// NewParallelUDPResolver creates a parallel DNS over UDP resolver.
	NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver
}
