package webconnectivity

import (
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/runtimex"
)

// EndpointInfo describes a TCP/TLS endpoint.
type EndpointInfo struct {
	String       string // String representation
	URLGetterURL string // URL for urlgetter
}

// EndpointsList is a list of EndpointInfo
type EndpointsList []EndpointInfo

// Endpoints returns a list of endpoints for TCP connect
func (el EndpointsList) Endpoints() (out []string) {
	out = []string{}
	for _, ei := range el {
		out = append(out, ei.String)
	}
	return
}

// URLs returns a list of URLs for TCP urlgetter
func (el EndpointsList) URLs() (out []string) {
	out = []string{}
	for _, ei := range el {
		out = append(out, ei.URLGetterURL)
	}
	return
}

// NewEndpoints creates a list of TCP/TLS endpoints to test from the
// target URL and the list of resolved IP addresses.
func NewEndpoints(URL *url.URL, addrs []string) (out EndpointsList) {
	out = EndpointsList{}
	port := NewEndpointPort(URL)
	for _, addr := range addrs {
		endpoint := net.JoinHostPort(addr, port.Port)
		out = append(out, EndpointInfo{
			String:       endpoint,
			URLGetterURL: (&url.URL{Scheme: port.URLGetterScheme, Host: endpoint}).String(),
		})
	}
	return
}

// EndpointPort is the port to be used by a TCP/TLS endpoint.
type EndpointPort struct {
	URLGetterScheme string
	Port            string
}

// NewEndpointPort creates an EndpointPort from the given URL. This function
// panic if the scheme is not `http` or `https` as well as if the host is not
// valid. The latter should not happen if you used url.Parse.
func NewEndpointPort(URL *url.URL) (out EndpointPort) {
	if URL.Scheme != "http" && URL.Scheme != "https" {
		panic("passed an unexpected scheme")
	}
	switch URL.Scheme {
	case "http":
		out.URLGetterScheme, out.Port = "tcpconnect", "80"
	case "https":
		out.URLGetterScheme, out.Port = "tlshandshake", "443"
	}
	if URL.Host != URL.Hostname() {
		_, port, err := net.SplitHostPort(URL.Host)
		runtimex.PanicOnError(err, "SplitHostPort should not fail here")
		out.Port = port
	}
	return
}
