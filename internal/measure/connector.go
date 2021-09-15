package measure

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Connector connects TCP (or UDP) connections.
type Connector interface {
	// Connect establishes a TCP (or UDP) connection. The address
	// argument must be a TCP or UDP endpoint address, i.e., an
	// IPv4 or quoted IPv6 address followed by ":" and by a port.
	Connect(ctx context.Context, network, address string) *ConnectResult
}

// ConnectResult contains the result of Connector.Connect.
type ConnectResult struct {
	// Network is the network we're connecting to.
	Network string `json:"network"`

	// Address is the address of the endpoint we're connecting to.
	Address string `json:"address"`

	// Started is when we started.
	Started time.Duration `json:"started"`

	// Completed is when we were done.
	Completed time.Duration `json:"completed"`

	// Failure contains the error or nil.
	Failure error `json:"failure"`

	// Addrs contains the established connection.
	Conn net.Conn `json:"-"`
}

// NewConnector creates a new Connector instance.
//
// The begin param is the beginning-of-time reference used to compute
// the elapsed time for several events.
//
// The logger param emits logs.
//
// The trace param collects a trace of I/O events.
//
// No param should be unset or nil.
func NewConnector(begin time.Time, logger Logger, trace *Trace) Connector {
	return &connector{begin: begin, logger: logger, trace: trace}
}

type connector struct {
	begin  time.Time
	logger Logger
	trace  *Trace
}

func (c *connector) Connect(
	ctx context.Context, network, address string) *ConnectResult {
	dialer := netxlite.NewDialerWithoutResolver(c.logger)
	defer dialer.CloseIdleConnections() // respect the protocol
	m := &ConnectResult{
		Network: network,
		Address: address,
		Started: time.Since(c.begin),
	}
	m.Conn, m.Failure = c.trace.dial(ctx, dialer, network, address)
	m.Completed = time.Since(c.begin)
	return m
}
