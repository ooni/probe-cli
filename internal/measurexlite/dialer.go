package measurexlite

//
// Dialer tracing
//

import (
	"context"
	"log"
	"math"
	"net"
	"strconv"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// NewDialerWithoutResolver is equivalent to netxlite.NewDialerWithoutResolver
// except that it returns a model.Dialer that uses this trace.
//
// Note: unlike code in netx or measurex, this factory DOES NOT return you a
// dialer that also performs wrapping of a net.Conn in case of success. If you
// want to wrap the conn, you need to wrap it explicitly using WrapNetConn.
func (tx *Trace) NewDialerWithoutResolver(dl model.DebugLogger) model.Dialer {
	return &dialerTrace{
		d:  tx.newDialerWithoutResolver(dl),
		tx: tx,
	}
}

// dialerTrace is a trace-aware model.Dialer.
type dialerTrace struct {
	d  model.Dialer
	tx *Trace
}

var _ model.Dialer = &dialerTrace{}

// DialContext implements model.Dialer.DialContext.
func (d *dialerTrace) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return d.d.DialContext(netxlite.WithTrace(ctx, d.tx), network, address)
}

// CloseIdleConnections implements model.Dialer.CloseIdleConnections.
func (d *dialerTrace) CloseIdleConnections() {
	d.d.CloseIdleConnections()
}

// OnTCPConnectDone implements model.Trace.OnTCPConnectDone.
func (tx *Trace) OnConnectDone(
	started time.Time, network, domain, remoteAddr string, err error, finished time.Time) {
	switch network {
	case "tcp", "tcp4", "tcp6":
		select {
		case tx.TCPConnect <- NewArchivalTCPConnectResult(
			tx.Index,
			started.Sub(tx.ZeroTime),
			remoteAddr,
			err,
			finished.Sub(tx.ZeroTime),
		):
		default: // buffer is full
		}
	default:
		// ignore UDP connect attempts because they cannot fail
		// in interesting ways that make sense for censorship
	}
}

// NewArchivalTCPConnectResult generates a model.ArchivalTCPConnectResult
// from the available information right after connect returns.
func NewArchivalTCPConnectResult(index int64, started time.Duration, address string,
	err error, finished time.Duration) *model.ArchivalTCPConnectResult {
	ip, port := archivalSplitHostPort(address)
	return &model.ArchivalTCPConnectResult{
		IP:   ip,
		Port: archivalPortToString(port),
		Status: model.ArchivalTCPConnectStatus{
			Blocked: nil,
			Failure: tracex.NewFailure(err),
			Success: err == nil,
		},
		T: finished.Seconds(),
	}
}

// archivalSplitHostPort is like net.SplitHostPort but does not return an error. This
// function returns two empty strings in case of any failure.
func archivalSplitHostPort(endpoint string) (string, string) {
	addr, port, err := net.SplitHostPort(endpoint)
	if err != nil {
		log.Printf("BUG: archivalSplitHostPort: invalid endpoint: %s", endpoint)
		return "", ""
	}
	return addr, port
}

// archivalPortToString is like strconv.Atoi but does not return an error. This
// function returns a zero port number in case of any failure.
func archivalPortToString(sport string) int {
	port, err := strconv.Atoi(sport)
	if err != nil || port < 0 || port > math.MaxUint16 {
		log.Printf("BUG: archivalStrconvAtoi: invalid port: %s", sport)
		return 0
	}
	return port
}

// TCPConnects drains the network events buffered inside the TCPConnect channel.
func (tx *Trace) TCPConnects() (out []*model.ArchivalTCPConnectResult) {
	for {
		select {
		case ev := <-tx.TCPConnect:
			out = append(out, ev)
		default:
			return // done
		}
	}
}
