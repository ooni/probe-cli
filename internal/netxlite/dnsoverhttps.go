package netxlite

//
// DNS-over-HTTPS transport
//

import (
	"bytes"
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSOverHTTPSTransport is a DNS-over-HTTPS DNSTransport.
type DNSOverHTTPSTransport struct {
	// Client is the MANDATORY http client to use.
	Client model.HTTPClient

	// URL is the MANDATORY URL of the DNS-over-HTTPS server.
	URL string

	// HostOverride is OPTIONAL and allows to override the
	// Host header sent in every request.
	HostOverride string
}

// NewDNSOverHTTPSTransport creates a new DNSOverHTTPSTransport instance.
//
// Arguments:
//
// - client in http.Client-like type (e.g., http.DefaultClient);
//
// - URL is the DoH resolver URL (e.g., https://1.1.1.1/dns-query).
func NewDNSOverHTTPSTransport(client model.HTTPClient, URL string) *DNSOverHTTPSTransport {
	return NewDNSOverHTTPSTransportWithHostOverride(client, URL, "")
}

// NewDNSOverHTTPSTransportWithHostOverride creates a new DNSOverHTTPSTransport
// with the given Host header override.
func NewDNSOverHTTPSTransportWithHostOverride(
	client model.HTTPClient, URL, hostOverride string) *DNSOverHTTPSTransport {
	return &DNSOverHTTPSTransport{Client: client, URL: URL, HostOverride: hostOverride}
}

// RoundTrip sends a query and receives a reply.
func (t *DNSOverHTTPSTransport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
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
	return ReadAllContext(ctx, resp.Body)
}

// RequiresPadding returns true for DoH according to RFC8467.
func (t *DNSOverHTTPSTransport) RequiresPadding() bool {
	return true
}

// Network returns the transport network, i.e., "doh".
func (t *DNSOverHTTPSTransport) Network() string {
	return "doh"
}

// Address returns the URL we're using for the DoH server.
func (t *DNSOverHTTPSTransport) Address() string {
	return t.URL
}

// CloseIdleConnections closes idle connections, if any.
func (t *DNSOverHTTPSTransport) CloseIdleConnections() {
	t.Client.CloseIdleConnections()
}

var _ model.DNSTransport = &DNSOverHTTPSTransport{}
