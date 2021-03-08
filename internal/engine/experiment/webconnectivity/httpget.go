package webconnectivity

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// HTTPGetConfig contains the config for HTTPGet
type HTTPGetConfig struct {
	Addresses []string
	Session   model.ExperimentSession
	TargetURL *url.URL
}

// TODO(bassosimone): we should normalize the timings

// HTTPGetResult contains the results of HTTPGet
type HTTPGetResult struct {
	TestKeys urlgetter.TestKeys
	Failure  *string
}

// TODO(bassosimone): Web Connectivity uses too much external testing
// and we should actually expose much less to the outside by using
// internal testing and by making _many_ functions private.

// HTTPGetNewDNSCache constructs the DNSCache option for HTTPGet
// by combining domain and addresses into a single string. As a
// corner case, if the domain equals the addresses _and_ the domain
// is an IP address, we return an empty string. This corner case
// corresponds to Web Connectivity inputs like https://1.1.1.1.
func HTTPGetNewDNSCache(domain, addresses string) string {
	if domain == addresses && net.ParseIP(addresses) != nil {
		return ""
	}
	return fmt.Sprintf("%s %s", domain, addresses)
}

// HTTPGet performs the HTTP/HTTPS part of Web Connectivity.
func HTTPGet(ctx context.Context, config HTTPGetConfig) (out HTTPGetResult) {
	addresses := strings.Join(config.Addresses, " ")
	if addresses == "" {
		// TODO(bassosimone): what to do in this case? We clearly
		// cannot fill the DNS cache...
		return
	}
	target := config.TargetURL.String()
	config.Session.Logger().Infof("GET %s...", target)
	domain := config.TargetURL.Hostname()
	result, err := urlgetter.Getter{
		Config: urlgetter.Config{
			DNSCache: HTTPGetNewDNSCache(domain, addresses),
		},
		Session: config.Session,
		Target:  target,
	}.Get(ctx)
	config.Session.Logger().Infof("GET %s... %+v", target, err)
	out.Failure = result.Failure
	out.TestKeys = result
	return
}
