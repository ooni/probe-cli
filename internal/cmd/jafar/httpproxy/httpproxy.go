// Package httpproxy contains a censoring HTTP proxy. This proxy will
// vet all the traffic and reply with 451 responses for a configurable
// set of offending Host headers in incoming requests.
package httpproxy

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

const product = "jafar/0.1.0"

// CensoringProxy is a censoring HTTP proxy
type CensoringProxy struct {
	keywords  []string
	transport http.RoundTripper
}

// NewCensoringProxy creates a new CensoringProxy instance using
// the specified list of keywords to censor. keywords is the list
// of keywords that trigger censorship if any of them appears in
// the Host header of a request. dnsNetwork and dnsAddress are
// settings to configure the upstream, non censored DNS.
func NewCensoringProxy(
	keywords []string, uncensored http.RoundTripper,
) *CensoringProxy {
	return &CensoringProxy{keywords: keywords, transport: uncensored}
}

var blockpage = []byte(`<html><head>
  <title>451 Unavailable For Legal Reasons</title>
</head><body>
  <center><h1>451 Unavailable For Legal Reasons</h1></center>
  <p>This content is not available in your jurisdiction.</p>
</body></html>
`)

// ServeHTTP serves HTTP requests
func (p *CensoringProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation note: use Via header to detect in a loose way
	// requests originated by us and directed to us
	if r.Header.Get("Via") != "" || r.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	for _, pattern := range p.keywords {
		if strings.Contains(r.Host, pattern) {
			w.WriteHeader(http.StatusUnavailableForLegalReasons)
			w.Write(blockpage)
			return
		}
	}
	r.Header.Add("Via", product) // see above
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Host:   r.Host,
		Scheme: "http",
	})
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Add("Via", product) // see above
		return nil
	}
	proxy.Transport = p.transport
	proxy.ServeHTTP(w, r)
}

// Start starts the censoring proxy.
func (p *CensoringProxy) Start(address string) (*http.Server, net.Addr, error) {
	server := &http.Server{Handler: p}
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, nil, err
	}
	go server.Serve(listener)
	return server, listener.Addr(), nil
}
