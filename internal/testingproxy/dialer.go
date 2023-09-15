package testingproxy

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// dialerWithAssertions ensures that we're dialing with the proxy address.
type dialerWithAssertions struct {
	// ExpectAddress is the expected IP address to dial
	ExpectAddress string

	// Dialer is the underlying dialer to use
	Dialer model.Dialer
}

var _ model.Dialer = &dialerWithAssertions{}

// CloseIdleConnections implements model.Dialer.
func (d *dialerWithAssertions) CloseIdleConnections() {
	d.Dialer.CloseIdleConnections()
}

// DialContext implements model.Dialer.
func (d *dialerWithAssertions) DialContext(ctx context.Context, network string, address string) (net.Conn, error) {
	// make sure the network is tcp
	const expectNetwork = "tcp"
	runtimex.Assert(
		network == expectNetwork,
		fmt.Sprintf("dialerWithAssertions: expected %s, got %s", expectNetwork, network),
	)
	log.Printf("dialerWithAssertions: verified that the network is %s as expected", expectNetwork)

	// make sure the IP address is the expected one
	ipAddr, _ := runtimex.Try2(net.SplitHostPort(address))
	runtimex.Assert(
		ipAddr == d.ExpectAddress,
		fmt.Sprintf("dialerWithAssertions: expected %s, got %s", d.ExpectAddress, ipAddr),
	)
	log.Printf("dialerWithAssertions: verified that the address is %s as expected", d.ExpectAddress)

	// now that we're sure we're using the proxy, we can actually dial
	return d.Dialer.DialContext(ctx, network, address)
}
