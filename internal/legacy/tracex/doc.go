// Package tracex performs measurements using tracing. To use tracing means
// that we'll wrap netx data types (e.g., a Dialer) with equivalent data types
// saving events into a Saver data struture. Then we will use the data types
// normally (e.g., call the Dialer's DialContet method and then use the
// resulting connection). When done, we will extract the trace containing
// all the events that occurred from the saver and process it to determine
// what happened during the measurement itself.
//
// This package is now frozen. Please, use measurexlite for new code. See
// https://github.com/ooni/probe-cli/blob/master/docs/design/dd-003-step-by-step.md
// for details about this.
package tracex
