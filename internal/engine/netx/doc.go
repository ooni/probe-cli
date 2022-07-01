// Package netx contains code to perform network measurements.
//
// This library derives from https://github.com/ooni/netx and contains
// the original code we wrote for performing measurements in Go. Over
// time, most of the original code has been refactored away inside:
//
// * model/netx.go: definition of interfaces and structs
//
// * netxlite: low-level network library
//
// * bytecounter: support for counting bytes sent and received
//
// * multierror: representing multiple errors using a single error
//
// * tracex: support for measuring using tracing
//
// This refactoring of netx (called "the netx pivot") has been described
// in https://github.com/ooni/probe-cli/pull/396. We described the
// design, implementation, and pain points of the pre-pivot netx library
// in https://github.com/ooni/probe-engine/issues/359. In turn,
// https://github.com/ooni/netx/blob/master/DESIGN.md contains the
// original design document for the netx library.
//
// Measuring using tracing means that we use ordinary stdlib-like
// objects such as model.Dialer and model.HTTPTransport. Then, we'll
// extract results from a tracex.Saver to determine the result of
// the measurement. The most notable user of this library is
// experiment/urlgetter, which implements a flexible URL-getting library.
//
// Tracing has its own set of limitations, so while we're still using
// it for implementing many experiments, we're also tinkering with
// step-by-step approaches where we break down operations in more basic
// building blocks, e.g., DNS resolution and fetching URL given an
// hostname, a protocol (e.g., QUIC or HTTPS), and an endpoint.
//
// While we're experimenting with alternative approaches, we also want
// to keep this library running and stable. New code will probably
// not be implemented here rather in step-by-step libraries.
//
// New experiments that can be written in terms of netxlite and tracex
// SHOULD NOT use netx. Existing experiment using netx MAY be rewritten
// using just netxlite and tracex when feasible.
//
// Additionally, new code that does not need to perform measurements
// SHOULD NOT use netx and SHOULD instead use netxlite.
//
// See docs/design/dd-002-nets.md in the probe-cli repository for
// the design document describing this package.
//
// This package is now frozen. Please, use measurexlite for new code. See
// https://github.com/ooni/probe-cli/blob/master/docs/design/dd-003-step-by-step.md
// for details about this.
package netx
