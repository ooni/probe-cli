package netxlite

//
// DNS-over-HTTPS transport
//

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSOverHTTPSTransport is a DNS-over-HTTPS DNSTransport.
type DNSOverHTTPSTransport struct {
	// Client is the MANDATORY http client to use.
	Client model.HTTPClient

	// Decoder is the MANDATORY DNSDecoder.
	Decoder model.DNSDecoder

	// URL is the MANDATORY URL of the DNS-over-HTTPS server.
	URL string

	// HostOverride is OPTIONAL and allows to override the
	// Host header sent in every request.
	HostOverride string
}

// NewUnwrappedDNSOverHTTPSTransport creates a new DNSOverHTTPSTransport
// instance that has not been wrapped yet.
//
// Arguments:
//
// - client is a model.HTTPClient type;
//
// - URL is the DoH resolver URL (e.g., https://dns.google/dns-query).
func NewUnwrappedDNSOverHTTPSTransport(client model.HTTPClient, URL string) *DNSOverHTTPSTransport {
	return NewUnwrappedDNSOverHTTPSTransportWithHostOverride(client, URL, "")
}

// NewUnwrappedDNSOverHTTPSTransportWithHostOverride creates a new DNSOverHTTPSTransport
// with the given Host header override. This instance has not been wrapped yet.
func NewUnwrappedDNSOverHTTPSTransportWithHostOverride(
	client model.HTTPClient, URL, hostOverride string) *DNSOverHTTPSTransport {
	return &DNSOverHTTPSTransport{
		Client:       client,
		Decoder:      &DNSDecoderMiekg{},
		URL:          URL,
		HostOverride: hostOverride,
	}
}

// RoundTrip sends a query and receives a reply.
func (t *DNSOverHTTPSTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	rawQuery, err := query.Bytes()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	req, err := http.NewRequest("POST", t.URL, bytes.NewReader(rawQuery))
	if err != nil {
		return nil, err
	}
	req.Host = t.HostOverride
	req.Header.Set("user-agent", model.HTTPHeaderUserAgent)
	req.Header.Set("content-type", "application/dns-message")
	resp, err := t.Client.Do(req.WithContext(ctx))
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
	const maxresponsesize = 1 << 20
	limitReader := io.LimitReader(resp.Body, maxresponsesize)
	rawResponse, err := ReadAllContext(ctx, limitReader)
	if err != nil {
		return nil, err
	}
	return t.Decoder.DecodeResponse(rawResponse, query)
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
