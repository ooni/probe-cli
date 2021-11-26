// Package quictesting contains code useful to test QUIC.
package quictesting

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Domain is the the domain we should be testing using QUIC.
const Domain = "www.cloudflare.com"

// Address is the address we should be testing using QUIC.
var Address string

// Endpoint returns the endpoint to test using QUIC by combining
// the Address variable with the given port.
func Endpoint(port string) string {
	return net.JoinHostPort(Address, port)
}

func init() {
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	reso := &net.Resolver{}
	addrs, err := reso.LookupHost(ctx, Domain)
	runtimex.PanicOnError(err, "reso.LookupHost failed")
	for _, addr := range addrs {
		if !strings.Contains(addr, ":") {
			Address = addr
			break
		}
	}
}
