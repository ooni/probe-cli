package torcontrolalgo

import "context"

// SignalShutdown sends the SIGNAL SHUTDOWN command.
func SignalShutdown(ctx context.Context, conn Conn) error {
	_, _ = conn.SendRecv(ctx, "SIGNAL SHUTDOWN")
	return nil
}
