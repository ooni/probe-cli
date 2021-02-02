package webconnectivity

import (
	"context"
	"fmt"
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
			DNSCache: fmt.Sprintf("%s %s", domain, addresses),
		},
		Session: config.Session,
		Target:  target,
	}.Get(ctx)
	config.Session.Logger().Infof("GET %s... %+v", target, err)
	out.Failure = result.Failure
	out.TestKeys = result
	return
}
