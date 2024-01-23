package torcontrolalgo

import (
	"context"
	"strings"
)

// Bootstrap performs the bootstrap and returns the related events. In case of
// error, the output may not be empty and may contain interesting events.
func Bootstrap(ctx context.Context, conn Conn) (output []string, err error) {
	// start emitting bootstrap status events
	if err := SetEvents(ctx, conn, EventStatusClient); err != nil {
		return nil, err
	}

	// enable the network to start a new bootstrap
	if err := SetConfEnableNetwork(ctx, conn); err != nil {
		return nil, err
	}

	// collect bootstrap until success or context timeout
Loop:
	for {
		select {
		case <-ctx.Done():
			break Loop

		case ev := <-conn.Notifications():
			if strings.HasPrefix(ev.EndReplyLine, "STATUS_CLIENT NOTICE BOOTSTRAP PROGRESS=100") {
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
