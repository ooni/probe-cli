package webconnectivity

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPGetConfig contains the config for HTTPGet
type HTTPGetConfig struct {
	Addresses []string
	Begin     time.Time
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

// HTTPGetMakeDNSCache constructs the DNSCache option for HTTPGet
// by combining domain and addresses into a single string. As a
// corner case, if the domain is an IP address, we return an empty
// string. This corner case corresponds to Web Connectivity
// inputs like https://1.1.1.1.
func HTTPGetMakeDNSCache(domain, addresses string) string {
	if net.ParseIP(domain) != nil {
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
		Begin: config.Begin,
		Config: urlgetter.Config{
			DNSCache: HTTPGetMakeDNSCache(domain, addresses),
		},
		Session: config.Session,
		Target:  target,
	}.Get(ctx)
	config.Session.Logger().Infof("GET %s... %+v", target, model.ErrorToStringOrOK(err))
	out.Failure = result.Failure
	out.TestKeys = result
	return
}
