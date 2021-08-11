package nwcth

import (
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type h3URL struct {
	*url.URL
	proto string
}

func getH3URL(resp *http.Response) *h3URL {
	URL := resp.Request.URL
	if URL == nil {
		return nil
	}
	h3Svc := parseAltSvc(resp, URL)
	if h3Svc == nil {
		return nil
	}
	quicURL, err := url.Parse(URL.String())
	runtimex.PanicOnError(err, "url.Parse failed")
	quicURL.Host = h3Svc.authority
	return &h3URL{URL: quicURL, proto: h3Svc.proto}
}

// parseAltSvc parses the Alt-Svc HTTP header for entries advertising the use of H3
func parseAltSvc(resp *http.Response, URL *url.URL) *altSvcH3 {
	// TODO(bassosimone,kelmenhorst): see if we can make this algorithm more robust.
	if URL.Scheme != "https" {
		return nil
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
				return &altSvcH3{authority: authority, proto: kv[0]}
			}
		}
	}
	return nil
}

type altSvcH3 struct {
	authority string
	proto     string
}
