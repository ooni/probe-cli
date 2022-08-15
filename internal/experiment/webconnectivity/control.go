package webconnectivity

import (
	"context"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Control issues a control request and saves the results
// inside of the experiment's TestKeys.
type Control struct {
	// Addresses contains the MANDATORY addresses we've looked up.
	Addresses []string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// TestKeys is MANDATORY and contains the TestKeys.
	TestKeys *TestKeys

	// Session is the MANDATORY session to use.
	Session model.ExperimentSession

	// THAddr is the MANDATORY TH's URL.
	THAddr string

	// URL is the MANDATORY URL we are measuring.
	URL *url.URL

	// WaitGroup is the MANDATORY wait group this task belongs to.
	WaitGroup *sync.WaitGroup
}

// Start starts this task in a background goroutine.
func (c *Control) Start(ctx context.Context) {
	c.WaitGroup.Add(1)
	go func() {
		defer c.WaitGroup.Done() // synchronize with the parent
		c.Run(ctx)
	}()
}

// Run runs this task until completion.
func (c *Control) Run(ctx context.Context) {
	// create a subcontext attached to a maximum timeout
	const timeout = 30 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// create control request
	var endpoints []string
	for _, address := range c.Addresses {
		if port := c.URL.Port(); port != "" {
			endpoints = append(endpoints, net.JoinHostPort(address, port))
			continue
		}
		endpoints = append(endpoints, net.JoinHostPort(address, "443"))
		endpoints = append(endpoints, net.JoinHostPort(address, "80"))
	}
	creq := &webconnectivity.ControlRequest{
		HTTPRequest: c.URL.String(),
		HTTPRequestHeaders: map[string][]string{
			"Accept":          {model.HTTPHeaderAccept},
			"Accept-Language": {model.HTTPHeaderAcceptLanguage},
			"User-Agent":      {model.HTTPHeaderUserAgent},
		},
		TCPConnect: endpoints,
	}
	c.TestKeys.SetControlRequest(creq)

	// create logger for this operation
	ol := measurexlite.NewOperationLogger(c.Logger, "control for %s", creq.HTTPRequest)

	// create an API client
	clnt := (&httpx.APIClientTemplate{
		BaseURL:    c.THAddr,
		HTTPClient: c.Session.DefaultHTTPClient(),
		Logger:     c.Logger,
		UserAgent:  c.Session.UserAgent(),
	}).WithBodyLogging().Build()

	// issue the control request and wait for the response
	var cresp webconnectivity.ControlResponse
	err := clnt.PostJSON(ctx, "/", creq, &cresp)
	if err != nil {
		// make sure error is wrapped
		err = netxlite.NewTopLevelGenericErrWrapper(err)
		c.TestKeys.SetControlFailure(err)
		ol.Stop(err)
		return
	}

	// on success, save the control response
	c.TestKeys.SetControl(&cresp)
	ol.Stop(nil)
}
