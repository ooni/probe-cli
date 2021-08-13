package nwcth

import (
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type h3URL struct {
	URL   *url.URL
	proto string
}

var ErrNoH3Location = errors.New("no h3 location found")

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
	entries := strings.Split(alt_svc, ",")
	for _, e := range entries {
		keyvalpairs := strings.Split(e, ";")
		for _, p := range keyvalpairs {
			p = strings.Replace(p, "\"", "", -1)
			kv := strings.Split(p, "=")
			if _, ok := supportedQUICVersions[kv[0]]; ok {
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

type altSvcH3 struct {
	authority string
	proto     string
}
