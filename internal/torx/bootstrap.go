package torx

//
// bootstrap.go - code to perform the tor bootstrap.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

// Bootstrap performs the bootstrap and returns the related events. In case of
// error, the output may not be empty and may contain interesting events.
func Bootstrap(ctx context.Context, conn ControlTransport) (output []string, err error) {
	// start emitting bootstrap status events
	if err := SetEvents(ctx, conn, EventStatusClient); err != nil {
		return nil, err
	}

	// enable the network to start a new bootstrap
	if err := SetConf(ctx, conn, NewKeyValuePair("DisableNetwork", optional.Some("0"))); err != nil {
		return nil, err
	}

	// TODO(bassosimone): maybe here we should check with GETINFO
	// whether status/circuit-established is good before attempting
	// to perform a bootstrap? Another option here is that we
	// first disable network and then re-enable network such that
	// we're sure that there's gonna be a bootstrap.
	//
	// According to the spec, we should be using "status/bootstrap-phase"
	// to learn the current bootstrap-phase.
	//
	// However, this partially conflicts with using tor as the
	// tool with which we're doing circumvention, I think, though
	// probably this issue needs to be solved at another level.

	// collect bootstrap until success or context timeout
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop

		case ev := <-conn.Notifications():
			if strings.HasPrefix(ev.EndReplyLine, "STATUS_CLIENT NOTICE CIRCUIT_ESTABLISHED") {
				output = append(output, ev.EndReplyLine)
				break Loop
			}
			output = append(output, ev.EndReplyLine)
		}
	}

	// stop emitting bootstrap events
	if err := SetEvents(ctx, conn); err != nil {
		return output, err
	}

	// handle the case where the bootstrap timed out
	if err := ctx.Err(); err != nil {
		return output, err
	}

	return output, nil
}
