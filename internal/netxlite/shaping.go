package netxlite

import "github.com/ooni/probe-cli/v3/internal/model"

// NewMaybeShapingDialer takes in input a model.Dialer and returns in output another
// model.Dialer that MAY dial connections with I/O shaping, depending on whether
// the user builds with or without the `-tags shaping` CLI flag.
//
// We typically use `-tags shaping` when running integration tests for dash and ndt7 to
// avoiod hammering m-lab servers from the very-fast GitHub CI servers.
//
// See https://github.com/ooni/probe/issues/2112 for extra context.
func NewMaybeShapingDialer(dialer model.Dialer) model.Dialer {
	return newMaybeShapingDialer(dialer)
}
