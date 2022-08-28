package portfiltering

//
// TCPConnect for portfiltering
//

import (
	"context"
	"math/rand"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// tcpPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tcpPingLoop(ctx context.Context, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *model.ArchivalTCPConnectResult) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	rand.Shuffle(len(Ports), func(i, j int) {
		Ports[i], Ports[j] = Ports[j], Ports[i]
	})
	// TODO(DecFox): Do we want to scan ports in parallel (using go routines) or in a
	// randomized sequential order?
	for i, port := range Ports {
		addr := net.JoinHostPort(address, port)
		m.tcpPingAsync(ctx, int64(i), zeroTime, logger, addr, out)
		<-ticker.C
	}
}

// tcpPingAsync performs a TCP ping and emits the result onto the out channel.
func (m *Measurer) tcpPingAsync(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string, out chan<- *model.ArchivalTCPConnectResult) {
	out <- m.tcpConnect(ctx, index, zeroTime, logger, address)
}

// tcpConnect performs a TCP connect and returns the result to the caller.
func (m *Measurer) tcpConnect(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string) *model.ArchivalTCPConnectResult {
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "TCPConnect #%d %s", index, address)
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	return trace.FirstTCPConnectOrNil()
}
