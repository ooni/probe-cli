// Package uncensored contains code used by Jafar to evade its own
// censorship efforts by taking alternate routes.
package uncensored

import (
	"context"
	"errors"
	"net"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Client is DNS, HTTP, and TCP client.
type Client struct {
	dnsClient     model.Resolver
	httpTransport model.HTTPTransport
	dialer        model.Dialer
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
		dnsClient:     configuration.DNSClient,
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

var _ model.Resolver = DefaultClient

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

// Network implements Resolver.Network
func (c *Client) Network() string {
	return c.dnsClient.Network()
}

var _ model.Dialer = DefaultClient

// DialContext implements Dialer.DialContext
func (c *Client) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return c.dialer.DialContext(ctx, network, address)
}

var _ model.HTTPTransport = DefaultClient

// CloseIdleConnections implement HTTPRoundTripper.CloseIdleConnections
func (c *Client) CloseIdleConnections() {
	c.dnsClient.CloseIdleConnections()
	c.httpTransport.CloseIdleConnections()
}

// RoundTrip implement HTTPRoundTripper.RoundTrip
func (c *Client) RoundTrip(req *http.Request) (*http.Response, error) {
	return c.httpTransport.RoundTrip(req)
}
