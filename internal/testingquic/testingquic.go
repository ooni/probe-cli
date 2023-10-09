// Package testingquic allows to retrieve the domain and endpoint to use
// for all the integration tests that use QUIC.
//
// See https://github.com/ooni/probe/issues/1873 for context.
package testingquic

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const domain = "www.cloudflare.com"

var (
	address  string
	initOnce sync.Once
)

func mustInit() {
	// create a context using a reasonable timeout
	const timeout = 10 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// instantiate the system resolver
	reso := &net.Resolver{}

	// perform the lookup and panic on failure
	addrs := runtimex.Try1(reso.LookupHost(ctx, domain))

	// use the first non IPv6 addr
	for _, addr := range addrs {
		if !strings.Contains(addr, ":") {
			address = addr
			return
		}
	}
}

// MustEndpoint returns a QUIC endpoint using this package's default address and then given port.
//
// This function PANICS if we cannot find out the IP addr we should use.
func MustEndpoint(port string) string {
	initOnce.Do(mustInit)
	return net.JoinHostPort(address, port)
}

// MustDomain returns the domain to use for QUIC testing.
//
// This function PANICS if we cannot find out the IP addr we should use.
func MustDomain() string {
	initOnce.Do(mustInit)
	return domain
}
