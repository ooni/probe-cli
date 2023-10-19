package pdsl

import "github.com/ooni/probe-cli/v3/internal/model"

// Trace is the abstraction that potentially allows to trace network operations
// and collect OONI observations. The [Runtime] is responsible for creating a
// [Trace] and the [Trace] actual behavior depends on the [Runtime]. For example,
// [NewMinimalRuntime] constructs a minimal [Runtime] creating [Trace] instances
// that do not actually collect OONI observations.
//
// This interface is such that [*netxlite.Netx] and [*measurexlite.Trace] implement it.
type Trace interface {
	// NewDialerWithoutResolver returns a dialer that saves observations into this trace.
	//
	// Caveat: the dialer wrappers are there to implement the model.MeasuringNetwork
	// interface, but they're not used by this function.
	NewDialerWithoutResolver(dl model.DebugLogger, wrappers ...model.DialerWrapper) model.Dialer

	// NewQUICDialerWithoutResolver is equivalent to
	// netxlite.NewQUICDialerWithoutResolver except that it returns a
	// model.QUICDialer that uses this trace.
	//
	// Caveat: the dialer wrappers are there to implement the
	// model.MeasuringNetwork interface, but they're not used by this function.
	NewQUICDialerWithoutResolver(listener model.UDPListener,
		dl model.DebugLogger, wrappers ...model.QUICDialerWrapper) model.QUICDialer

	// NewStdlibResolver returns a resolver that saves observations into this trace.
	NewStdlibResolver(logger model.DebugLogger) model.Resolver

	// NewParallelUDPResolver creates a parallel DNS over UDP resolver.
	NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver

	// NewTLSHandshakerStdlib creates a TLS handshaker using the standard library.
	NewTLSHandshakerStdlib(dl model.DebugLogger) model.TLSHandshaker
}
