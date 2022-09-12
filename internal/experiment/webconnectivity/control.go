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

// EndpointMeasurementsStarter is used by Control to start extra
// measurements using new IP addrs discovered by the TH.
type EndpointMeasurementsStarter interface {
	// startCleartextFlows starts a TCP measurement flow for each IP addr. The [ps]
	// argument determines whether this flow will be allowed to fetch the webpage.
	startCleartextFlows(ctx context.Context, ps *prioritySelector, addresses []DNSEntry)

	// startSecureFlows is like startCleartextFlows but for HTTPS.
	startSecureFlows(ctx context.Context, ps *prioritySelector, addresses []DNSEntry)
}

// Control issues a Control request and saves the results
// inside of the experiment's TestKeys.
//
// The zero value of this structure IS NOT valid and you MUST initialize
// all the fields marked as MANDATORY before using this structure.
type Control struct {
	// Addresses contains the MANDATORY addresses we've looked up.
	Addresses []string

	// ExtraMeasurementsStarter is MANDATORY and allows this struct to
	// start additional measurements using new TH-discovered addrs.
	ExtraMeasurementsStarter EndpointMeasurementsStarter

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// PrioSelector is the OPTIONAL priority selector to use to determine
	// whether we will be allowed to fetch the webpage.
	PrioSelector *prioritySelector

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
func (c *Control) Run(parentCtx context.Context) {
	// create a subcontext attached to a maximum timeout
	const timeout = 30 * time.Second
	opCtx, cancel := context.WithTimeout(parentCtx, timeout)
	defer cancel()

	// create control request
	var endpoints []string
	for _, address := range c.Addresses {
		if port := c.URL.Port(); port != "" { // handle the case of a custom port
			endpoints = append(endpoints, net.JoinHostPort(address, port))
			continue
		}
		// otherwise, always attempt to measure both 443 and 80 endpoints
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
	ol := measurexlite.NewOperationLogger(
		c.Logger,
		"control for %s using %s",
		creq.HTTPRequest,
		c.THAddr,
	)

	// create an API client
	clnt := (&httpx.APIClientTemplate{
		Accept:        "",
		Authorization: "",
		BaseURL:       c.THAddr,
		HTTPClient:    c.Session.DefaultHTTPClient(),
		Host:          "", // use the one inside the URL
		LogBody:       true,
		Logger:        c.Logger,
		UserAgent:     c.Session.UserAgent(),
	}).Build()

	// issue the control request and wait for the response
	var cresp webconnectivity.ControlResponse
	err := clnt.PostJSON(opCtx, "/", creq, &cresp)
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

	// if the TH returned us addresses we did not previously were
	// aware of, make sure we also measure them
	c.maybeStartExtraMeasurements(parentCtx, cresp.DNS.Addrs)
}

// This function determines whether we should start new
// background measurements for previously unknown IP addrs.
func (c *Control) maybeStartExtraMeasurements(ctx context.Context, thAddrs []string) {
	// classify addeesses by who discovered them
	const (
		inProbe = 1 << iota
		inTH
	)
	mapping := make(map[string]int)
	for _, addr := range c.Addresses {
		mapping[addr] |= inProbe
	}
	for _, addr := range thAddrs {
		mapping[addr] |= inTH
	}

	// obtain the TH-only addresses
	var thOnlyAddrs []string
	for addr, flags := range mapping {
		if (flags & inProbe) != 0 {
			continue // discovered by the probe => already tested
		}
		thOnlyAddrs = append(thOnlyAddrs, addr)
	}

	c.Logger.Infof("measuring additional addrs from TH: %+v", thOnlyAddrs)

	var thOnly []DNSEntry
	for _, addr := range thOnlyAddrs {
		thOnly = append(thOnly, DNSEntry{
			Addr:  addr,
			Flags: 0, // neither system, nor udp, nor doh
		})
	}

	// Start extra measurements for TH-only addresses.
	c.ExtraMeasurementsStarter.startCleartextFlows(ctx, c.PrioSelector, thOnly)
	c.ExtraMeasurementsStarter.startSecureFlows(ctx, c.PrioSelector, thOnly)
}
