package portfiltering

//
// TCPConnect for portfiltering
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// tcpConnect performs a TCP connect and returns the result to the caller.
func (m *Measurer) tcpConnect(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, tk *TestKeys, address string) {
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "TCPConnect #%d %s", index, address)
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	tk.TCPConnect = trace.FirstTCPConnectOrNil()
}
