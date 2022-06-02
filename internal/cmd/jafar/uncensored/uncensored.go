// Package uncensored contains code used by Jafar to evade its own
// censorship efforts by taking alternate routes.
package uncensored

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Client is DNS, HTTP, and TCP client.
type Client struct {
	dnsClient     model.Resolver
	httpTransport model.HTTPTransport
	dialer        model.Dialer
}

// NewClient creates a new Client.
func NewClient(resolverURL string) *Client {
	dnsClient := netxlite.NewParallelDNSOverHTTPSResolver(log.Log, resolverURL)
	return &Client{
		dnsClient:     dnsClient,
		httpTransport: netxlite.NewHTTPTransportWithResolver(log.Log, dnsClient),
		dialer:        netxlite.NewDialerWithResolver(log.Log, dnsClient),
	}
}

// Address implements Resolver.Address
func (c *Client) Address() string {
	return c.dnsClient.Address()
}

// LookupHost implements Resolver.LookupHost
func (c *Client) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return c.dnsClient.LookupHost(ctx, domain)
}

// LookupHTTPS implements model.Resolver.LookupHTTPS.
func (c *Client) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, errors.New("not implemented")
}

// LookupNS implements model.Resolver.LookupNS.
func (c *Client) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, errors.New("not implemented")
}

// Network implements Resolver.Network
func (c *Client) Network() string {
	return c.dnsClient.Network()
}

// DialContext implements Dialer.DialContext
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return c.dialer.DialContext(ctx, network, address)
}

// CloseIdleConnections implement HTTPRoundTripper.CloseIdleConnections
func (c *Client) CloseIdleConnections() {
	c.dnsClient.CloseIdleConnections()
	c.httpTransport.CloseIdleConnections()
}

// RoundTrip implement HTTPRoundTripper.RoundTrip
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.httpTransport.RoundTrip(req)
}
