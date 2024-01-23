package torx

import "context"

// SignalShutdown sends the SIGNAL SHUTDOWN command.
func SignalShutdown(ctx context.Context, conn ControlTransport) error {
	_, _ = conn.SendRecv(ctx, "SIGNAL SHUTDOWN")
	return nil
}
