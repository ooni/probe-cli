package dslvm

import (
	"context"
	"net"
)

// MakeEndpointsStage is a [Stage] that transforms IP addresses to TCP/UDP endpoints.
type MakeEndpointsStage struct {
	// Input contains the MANDATORY channel from which to read IP addresses. We
	// assume that this channel will be closed when done.
	Input <-chan string

	// Output is the MANDATORY channel emitting endpoints. We will close this
	// channel when the Input channel has been closed.
	Output chan<- string

	// Port is the MANDATORY port.
	Port string
}

var _ Stage = &MakeEndpointsStage{}

// Run transforms IP addresses to endpoints.
func (sx *MakeEndpointsStage) Run(ctx context.Context, rtx Runtime) {
	defer close(sx.Output)
	for addr := range sx.Input {
		sx.Output <- net.JoinHostPort(addr, sx.Port)
	}
}
