package torx

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// TakeOwnership sends the TAKEOWNERSHIP command.
func TakeOwnership(ctx context.Context, conn ControlTransport) error {
	resp, err := conn.SendRecv(ctx, "TAKEOWNERSHIP")
	if err != nil {
		return err
	}
	if resp.Status != torcontrolnet.StatusOk {
		return ErrControlRequestFailed
	}
	return nil
}
