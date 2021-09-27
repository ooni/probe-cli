package dnsx

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

// HTTPClient is the HTTP client expected by DNSOverHTTPS.
type HTTPClient interface {
	Do(req *http.Request) (*http.Response, error)
	CloseIdleConnections()
}

// DNSOverHTTPS is a DNS over HTTPS RoundTripper. Requests are submitted over
// an HTTP/HTTPS channel provided by URL using the Do function.
type DNSOverHTTPS struct {
	Client       HTTPClient
	URL          string
	HostOverride string
}

// NewDNSOverHTTPS creates a new DNSOverHTTP instance from the
// specified http.Client and URL, as a convenience.
func NewDNSOverHTTPS(client HTTPClient, URL string) *DNSOverHTTPS {
	return NewDNSOverHTTPSWithHostOverride(client, URL, "")
}

// NewDNSOverHTTPSWithHostOverride is like NewDNSOverHTTPS except that
// it's creating a resolver where we use the specified host.
func NewDNSOverHTTPSWithHostOverride(
	client HTTPClient, URL, hostOverride string) *DNSOverHTTPS {
	return &DNSOverHTTPS{Client: client, URL: URL, HostOverride: hostOverride}
}

// RoundTrip implements RoundTripper.RoundTrip.
func (t *DNSOverHTTPS) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequest("POST", t.URL, bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Host = t.HostOverride
	req.Header.Set("user-agent", httpheader.UserAgent())
	req.Header.Set("content-type", "application/dns-message")
	var resp *http.Response
	resp, err = t.Client.Do(req.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// TODO(bassosimone): we should map the status code to a
		// proper Error in the DNS context.
		return nil, errors.New("doh: server returned error")
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		return nil, errors.New("doh: invalid content-type")
	}
	return iox.ReadAllContext(ctx, resp.Body)
}

// RequiresPadding returns true for DoH according to RFC8467
func (t *DNSOverHTTPS) RequiresPadding() bool {
	return true
}

// Network returns the transport network (e.g., doh, dot)
func (t *DNSOverHTTPS) Network() string {
	return "doh"
}

// Address returns the upstream server address.
func (t *DNSOverHTTPS) Address() string {
	return t.URL
}

// CloseIdleConnections closes idle connections.
func (t *DNSOverHTTPS) CloseIdleConnections() {
	t.Client.CloseIdleConnections()
}

var _ DNSTransport = &DNSOverHTTPS{}
