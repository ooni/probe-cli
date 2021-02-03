package resolver

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
)

// DNSOverHTTPS is a DNS over HTTPS RoundTripper. Requests are submitted over
// an HTTP/HTTPS channel provided by URL using the Do function.
type DNSOverHTTPS struct {
	Do           func(req *http.Request) (*http.Response, error)
	URL          string
	HostOverride string
}

// NewDNSOverHTTPS creates a new DNSOverHTTP instance from the
// specified http.Client and URL, as a convenience.
func NewDNSOverHTTPS(client *http.Client, URL string) DNSOverHTTPS {
	return NewDNSOverHTTPSWithHostOverride(client, URL, "")
}

// NewDNSOverHTTPSWithHostOverride is like NewDNSOverHTTPS except that
// it's creating a resolver where we use the specified host.
func NewDNSOverHTTPSWithHostOverride(client *http.Client, URL, hostOverride string) DNSOverHTTPS {
	return DNSOverHTTPS{Do: client.Do, URL: URL, HostOverride: hostOverride}
}

// RoundTrip implements RoundTripper.RoundTrip.
func (t DNSOverHTTPS) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
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
	resp, err = t.Do(req.WithContext(ctx))
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
	return ioutil.ReadAll(resp.Body)
}

// RequiresPadding returns true for DoH according to RFC8467
func (t DNSOverHTTPS) RequiresPadding() bool {
	return true
}

// Network returns the transport network (e.g., doh, dot)
func (t DNSOverHTTPS) Network() string {
	return "doh"
}

// Address returns the upstream server address.
func (t DNSOverHTTPS) Address() string {
	return t.URL
}

var _ RoundTripper = DNSOverHTTPS{}
