package websteps

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/websteps"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type h3URL struct {
	URL   *url.URL
	proto string
}

type altSvcH3 struct {
	authority string
	proto     string
}

var ErrNoH3Location = errors.New("no h3 location found")

// getH3URL returns the URL for HTTP/3 requests, if the target supports HTTP/3.
// Returns nil, if no HTTP/3 support is advertised.
func getH3URL(resp *http.Response) (*h3URL, error) {
	URL := resp.Request.URL
	if URL == nil {
		return nil, ErrInvalidURL
	}
	h3Svc, err := parseAltSvc(resp, URL)
	if err != nil {
		return nil, err
	}
	quicURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	quicURL.Host = h3Svc.authority
	return &h3URL{URL: quicURL, proto: h3Svc.proto}, nil
}

// parseAltSvc parses the Alt-Svc HTTP header for entries advertising the use of H3
func parseAltSvc(resp *http.Response, URL *url.URL) (*altSvcH3, error) {
	// TODO(bassosimone,kelmenhorst): see if we can make this algorithm more robust.
	if URL.Scheme != "https" {
		return nil, ErrUnsupportedScheme
	}
	alt_svc := resp.Header.Get("Alt-Svc")
	// syntax: Alt-Svc: <protocol-id>=<alt-authority>; ma=<max-age>; persist=1
	entries := strings.Split(alt_svc, ",")
	for _, e := range entries {
		keyvalpairs := strings.Split(e, ";")
		for _, p := range keyvalpairs {
			p = strings.Replace(p, "\"", "", -1)
			kv := strings.Split(p, "=")
			if len(kv) != 2 {
				continue
			}
			if _, ok := websteps.SupportedQUICVersions[kv[0]]; ok {
				host, port, err := net.SplitHostPort(kv[1])
				runtimex.PanicOnError(err, "net.SplitHostPort failed")
				if host == "" {
					host = URL.Hostname()
				}
				authority := net.JoinHostPort(host, port)
				return &altSvcH3{authority: authority, proto: kv[0]}, nil
			}
		}
	}
	return nil, ErrNoH3Location
}
