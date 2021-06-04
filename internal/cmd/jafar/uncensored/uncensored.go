// Package uncensored contains code used by Jafar to evade its own
// censorship efforts by taking alternate routes.
package uncensored

import (
	"context"
	"net"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Client is DNS, HTTP, and TCP client.
type Client struct {
	dnsClient     *netx.DNSClient
	httpTransport netx.HTTPRoundTripper
	dialer        netx.Dialer
}

// NewClient creates a new Client.
func NewClient(resolverURL string) (*Client, error) {
	configuration, err := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: resolverURL,
		},
		Logger: log.Log,
	}.NewConfiguration()
	if err != nil {
		return nil, err
	}
	return &Client{
		dnsClient:     &configuration.DNSClient,
		httpTransport: netx.NewHTTPTransport(configuration.HTTPConfig),
		dialer:        netx.NewDialer(configuration.HTTPConfig),
	}, nil
}

// Must panics if it's not possible to create a Client. Usually you should
// use it like `uncensored.Must(uncensored.NewClient(URL))`.
func Must(client *Client, err error) *Client {
	runtimex.PanicOnError(err, "cannot create uncensored client")
	return client
}

// DefaultClient is the default client for DNS, HTTP, and TCP.
var DefaultClient = Must(NewClient(""))

var _ netx.Resolver = DefaultClient

// Address implements netx.Resolver.Address
func (c *Client) Address() string {
	return c.dnsClient.Address()
}

// LookupHost implements netx.Resolver.LookupHost
func (c *Client) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return c.dnsClient.LookupHost(ctx, domain)
}

// Network implements netx.Resolver.Network
func (c *Client) Network() string {
	return c.dnsClient.Network()
}

var _ netx.Dialer = DefaultClient

// DialContext implements netx.Dialer.DialContext
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return c.dialer.DialContext(ctx, network, address)
}

var _ netx.HTTPRoundTripper = DefaultClient

// CloseIdleConnections implement netx.HTTPRoundTripper.CloseIdleConnections
func (c *Client) CloseIdleConnections() {
	c.dnsClient.CloseIdleConnections()
	c.httpTransport.CloseIdleConnections()
}

// RoundTrip implement netx.HTTPRoundTripper.RoundTrip
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.httpTransport.RoundTrip(req)
}
