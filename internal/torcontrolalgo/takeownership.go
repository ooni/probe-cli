package torcontrolalgo

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// TakeOwnership sends the TAKEOWNERSHIP command.
func TakeOwnership(ctx context.Context, conn Conn) error {
	resp, err := conn.SendRecv(ctx, "TAKEOWNERSHIP")
	if err != nil {
		return err
	}
	if resp.Status != torcontrolnet.StatusOk {
		return ErrRequestFailed
	}
	return nil
}
