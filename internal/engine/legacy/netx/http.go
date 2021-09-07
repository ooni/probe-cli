package netx

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/oldhttptransport"
	errorsxlegacy "github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"golang.org/x/net/http2"
)

// HTTPTransport performs single HTTP transactions and emits
// measurement events as they happen.
type HTTPTransport struct {
	Beginning    time.Time
	Dialer       *Dialer
	Handler      modelx.Handler
	Transport    *http.Transport
	roundTripper http.RoundTripper
}

func newHTTPTransport(
	beginning time.Time,
	handler modelx.Handler,
	dialer *Dialer,
	disableKeepAlives bool,
	proxyFunc func(*http.Request) (*url.URL, error),
) *HTTPTransport {
	baseTransport := &http.Transport{
		// The following values are copied from Go 1.12 docs and match
		// what should be used by the default transport
		ExpectContinueTimeout: 1 * time.Second,
		IdleConnTimeout:       90 * time.Second,
		MaxIdleConns:          100,
		Proxy:                 proxyFunc,
		TLSHandshakeTimeout:   10 * time.Second,
		DisableKeepAlives:     disableKeepAlives,
	}
	ooniTransport := oldhttptransport.New(baseTransport)
	// Configure h2 and make sure that the custom TLSConfig we use for dialing
	// is actually compatible with upgrading to h2. (This mainly means we
	// need to make sure we include "h2" in the NextProtos array.) Because
	// http2.ConfigureTransport only returns error when we have already
	// configured http2, it is safe to ignore the return value.
	http2.ConfigureTransport(baseTransport)
	// Since we're not going to use our dialer for TLS, the main purpose of
	// the following line is to make sure ForseSpecificSNI has impact on the
	// config we are going to use when doing TLS. The code is as such since
	// we used to force net/http through using dialer.DialTLS.
	dialer.TLSConfig = baseTransport.TLSClientConfig
	// Arrange the configuration such that we always use `dialer` for dialing
	// cleartext connections. The net/http code will dial TLS connections.
	baseTransport.DialContext = dialer.DialContext
	// Better for Cloudflare DNS and also better because we have less
	// noisy events and we can better understand what happened.
	baseTransport.MaxConnsPerHost = 1
	// The following (1) reduces the number of headers that Go will
	// automatically send for us and (2) ensures that we always receive
	// back the true headers, such as Content-Length. This change is
	// functional to OONI's goal of observing the network.
	baseTransport.DisableCompression = true
	return &HTTPTransport{
		Beginning:    beginning,
		Dialer:       dialer,
		Handler:      handler,
		Transport:    baseTransport,
		roundTripper: ooniTransport,
	}
}

// RoundTrip executes a single HTTP transaction, returning
// a Response for the provided Request.
func (t *HTTPTransport) RoundTrip(
	req *http.Request,
) (resp *http.Response, err error) {
	ctx := maybeWithMeasurementRoot(req.Context(), t.Beginning, t.Handler)
	req = req.WithContext(ctx)
	resp, err = t.roundTripper.RoundTrip(req)
	// For safety wrap the error as modelx.HTTPRoundTripOperation but this
	// will only be used if the error chain does not contain any
	// other major operation failure. See errorsx.ErrWrapper.
	err = errorsxlegacy.SafeErrWrapperBuilder{
		Error:     err,
		Operation: errorsx.HTTPRoundTripOperation,
	}.MaybeBuild()
	return resp, err
}

// CloseIdleConnections closes the idle connections.
func (t *HTTPTransport) CloseIdleConnections() {
	// Adapted from net/http code
	type closeIdler interface {
		CloseIdleConnections()
	}
	if tr, ok := t.roundTripper.(closeIdler); ok {
		tr.CloseIdleConnections()
	}
}

// NewHTTPTransportWithProxyFunc creates a transport without any
// handler attached using the specified proxy func.
func NewHTTPTransportWithProxyFunc(
	proxyFunc func(*http.Request) (*url.URL, error),
) *HTTPTransport {
	return newHTTPTransport(time.Now(), handlers.NoHandler, NewDialer(), false, proxyFunc)
}

// NewHTTPTransport creates a new HTTP transport.
func NewHTTPTransport() *HTTPTransport {
	return NewHTTPTransportWithProxyFunc(http.ProxyFromEnvironment)
}

// ConfigureDNS is exactly like netx.Dialer.ConfigureDNS.
func (t *HTTPTransport) ConfigureDNS(network, address string) error {
	return t.Dialer.ConfigureDNS(network, address)
}

// SetResolver is exactly like netx.Dialer.SetResolver.
func (t *HTTPTransport) SetResolver(r modelx.DNSResolver) {
	t.Dialer.SetResolver(r)
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (t *HTTPTransport) SetCABundle(path string) error {
	return t.Dialer.SetCABundle(path)
}

// ForceSpecificSNI forces using a specific SNI.
func (t *HTTPTransport) ForceSpecificSNI(sni string) error {
	return t.Dialer.ForceSpecificSNI(sni)
}

// ForceSkipVerify forces to skip certificate verification
func (t *HTTPTransport) ForceSkipVerify() error {
	return t.Dialer.ForceSkipVerify()
}

// HTTPClient is a replacement for http.HTTPClient.
type HTTPClient struct {
	// HTTPClient is the underlying client. Pass this client to existing code
	// that expects an *http.HTTPClient. For this reason we can't embed it.
	HTTPClient *http.Client

	// Transport is the transport configured by NewClient to be used
	// by the HTTPClient field.
	Transport *HTTPTransport
}

// NewHTTPClientWithProxyFunc creates a new client using the
// specified proxyFunc for handling proxying.
func NewHTTPClientWithProxyFunc(
	proxyFunc func(*http.Request) (*url.URL, error),
) *HTTPClient {
	transport := NewHTTPTransportWithProxyFunc(proxyFunc)
	return &HTTPClient{
		HTTPClient: &http.Client{Transport: transport},
		Transport:  transport,
	}
}

// NewHTTPClient creates a new client instance.
func NewHTTPClient() *HTTPClient {
	return NewHTTPClientWithProxyFunc(http.ProxyFromEnvironment)
}

// NewHTTPClientWithoutProxy creates a new client instance that
// does not use any kind of proxy.
func NewHTTPClientWithoutProxy() *HTTPClient {
	return NewHTTPClientWithProxyFunc(nil)
}

// ConfigureDNS internally calls netx.Dialer.ConfigureDNS and
// therefore it has the same caveats and limitations.
func (c *HTTPClient) ConfigureDNS(network, address string) error {
	return c.Transport.ConfigureDNS(network, address)
}

// SetResolver internally calls netx.Dialer.SetResolver
func (c *HTTPClient) SetResolver(r modelx.DNSResolver) {
	c.Transport.SetResolver(r)
}

// SetCABundle internally calls netx.Dialer.SetCABundle and
// therefore it has the same caveats and limitations.
func (c *HTTPClient) SetCABundle(path string) error {
	return c.Transport.SetCABundle(path)
}

// ForceSpecificSNI forces using a specific SNI.
func (c *HTTPClient) ForceSpecificSNI(sni string) error {
	return c.Transport.ForceSpecificSNI(sni)
}

// ForceSkipVerify forces to skip certificate verification
func (c *HTTPClient) ForceSkipVerify() error {
	return c.Transport.ForceSkipVerify()
}

// CloseIdleConnections closes the idle connections.
func (c *HTTPClient) CloseIdleConnections() {
	c.Transport.CloseIdleConnections()
}
