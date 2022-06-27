package tlsmiddlebox

import (
	"context"
	"net"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// TCPConnect connects to the endpoint and sends the model.ArchivalTCPConnectResult to the buffered channel
func (m *Measurer) MeasureTCP(ctx context.Context, addr string, tcpEvents chan<- *model.ArchivalTCPConnectResult) error {
	logger := model.ValidLoggerOrDefault(nil)
	dialer := netxlite.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	out := writeTCPtoArchival(addr, err)
	if conn != nil {
		conn.Close()
	}
	select {
	case tcpEvents <- out:
	default:
	}
	return err
}

// writeTCPtoArchival writes the TCPConnect results to model.ArchivalTCPConnectResult
// while we may receive an errWrapper in cerr, we do not generate a failure here
// and rather add this error directly to the measurement
func writeTCPtoArchival(addr string, cerr error) (out *model.ArchivalTCPConnectResult) {
	var failure *string
	var errStr string
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		errStr = err.Error()
		failure = &errStr
		out = &model.ArchivalTCPConnectResult{
			Status: model.ArchivalTCPConnectStatus{
				Failure: failure,
				Success: false,
			},
		}
		return
	}
	p, err := strconv.Atoi(port)
	if err != nil {
		errStr = err.Error()
		failure = &errStr
		out = &model.ArchivalTCPConnectResult{
			Status: model.ArchivalTCPConnectStatus{
				Failure: failure,
				Success: false,
			},
		}
		return
	}
	if cerr != nil {
		errStr = cerr.Error()
		failure = &errStr
	}
	out = &model.ArchivalTCPConnectResult{
		IP:   host,
		Port: int(p),
		Status: model.ArchivalTCPConnectStatus{
			Failure: failure,
			Success: (cerr == nil),
		},
	}
	return
}

// GetTCPEvents receives the tcpEvents from the buffered channel and adds it to an output array
func GetTCPEvents(tcpEvents <-chan *model.ArchivalTCPConnectResult) (out []*model.ArchivalTCPConnectResult) {
	for {
		select {
		case ev := <-tcpEvents:
			out = append(out, ev)
		default:
			return
		}
	}
}
