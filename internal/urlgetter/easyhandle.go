package urlgetter

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// EasyHandle measures URLs sequentially.
//
// The zero value is invalid. Please, initialize the MANDATORY fields.
type EasyHandle struct {
	// Begin is the OPTIONAL time when the experiment begun. If you do not
	// set this field, every target is measured independently.
	Begin time.Time

	// IndexGen is the MANDATORY index generator.
	IndexGen RunnerTraceIndexGenerator

	// Session is the MANDATORY session to use. If this is nil, the Run
	// method will panic with a nil pointer error.
	Session RunnerSession

	// UNet is the OPTIONAL underlying network to use.
	UNet model.UnderlyingNetwork
}

// EasyTarget is a target for [*EasyHandle].
type EasyTarget struct {
	// Config contains the target configuration.
	Config *Config

	// URL contains the URL to measure.
	URL string
}

// Run gets the target URL and returns either the [*TestKeys] or an error.
func (hx *EasyHandle) Run(ctx context.Context, target *EasyTarget) (*TestKeys, error) {
	// parse the target URL
	URL, err := url.Parse(target.URL)

	// handle the case where we cannot parse the URL.
	if err != nil {
		return nil, fmt.Errorf("urlgetter: invalid target URL: %w", err)
	}

	// obtain the measurement zero time
	begin := hx.Begin
	if begin.IsZero() {
		begin = time.Now()
	}

	// create the test keys
	tk := &TestKeys{
		Agent:           "",
		BootstrapTime:   0,
		DNSCache:        []string{},
		FailedOperation: optional.None[string](),
		Failure:         optional.None[string](),
		NetworkEvents:   []*model.ArchivalNetworkEvent{},
		Queries:         []*model.ArchivalDNSLookupResult{},
		Requests:        []*model.ArchivalHTTPRequestResult{},
		SOCKSProxy:      "",
		TCPConnect:      []*model.ArchivalTCPConnectResult{},
		TLSHandshakes:   []*model.ArchivalTLSOrQUICHandshakeResult{},
		Tunnel:          "",
	}

	// create the runner
	runner := &Runner{
		Begin:    begin,
		IndexGen: hx.IndexGen,
		Session:  hx.Session,
		TestKeys: tk,
		UNet:     hx.UNet,
	}

	// measure using the runner
	err = runner.Run(ctx, target.Config, URL)

	// handle the case of fundamental measurement failure
	if err != nil {
		return nil, err
	}

	// otherwise return the test keys
	return tk, nil
}
