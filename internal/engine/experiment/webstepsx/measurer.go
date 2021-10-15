package webstepsx

//
// Measurer
//
// This file contains the client implementation.
//

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "websteps"
	testVersion = "0.0.2"
)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment's test keys.
type TestKeys struct {
	*measurex.URLMeasurement
}

// Measurer performs the measurement.
type Measurer struct {
	Config Config
}

var (
	_ model.ExperimentMeasurer      = &Measurer{}
	_ model.ExperimentMeasurerAsync = &Measurer{}
)

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{Config: config}
}

// ExperimentName implements ExperimentMeasurer.ExperExperimentName.
func (mx *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperExperimentVersion.
func (mx *Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// ErrNoAvailableTestHelpers is emitted when there are no available test helpers.
	ErrNoAvailableTestHelpers = errors.New("no available helpers")

	// ErrNoInput indicates that no input was provided.
	ErrNoInput = errors.New("no input provided")

	// ErrInputIsNotAnURL indicates that the input is not an URL.
	ErrInputIsNotAnURL = errors.New("input is not an URL")

	// ErrUnsupportedInput indicates that the input URL scheme is unsupported.
	ErrUnsupportedInput = errors.New("unsupported input scheme")
)

// RunAsync implements ExperimentMeasurerAsync.RunAsync.
func (mx *Measurer) RunAsync(
	ctx context.Context, sess model.ExperimentSession, input string,
	callbacks model.ExperimentCallbacks) (<-chan *model.ExperimentAsyncTestKeys, error) {
	// 1. Parse and verify URL
	URL, err := url.Parse(input)
	if err != nil {
		return nil, ErrInputIsNotAnURL
	}
	if URL.Scheme != "http" && URL.Scheme != "https" {
		return nil, ErrUnsupportedInput
	}
	// 2. Find the testhelper
	testhelpers, _ := sess.GetTestHelpersByName("web-connectivity")
	var testhelper *model.Service
	for _, th := range testhelpers {
		if th.Type == "https" {
			testhelper = &th
			break
		}
	}
	if testhelper == nil {
		return nil, ErrNoAvailableTestHelpers
	}
	testhelper.Address = "https://1.th.ooni.org/api/v1/websteps" // TODO(bassosimone): remove!
	out := make(chan *model.ExperimentAsyncTestKeys)
	go mx.runAsync(ctx, sess, input, testhelper, out)
	return out, nil
}

var measurerResolvers = []*measurex.ResolverInfo{{
	Network: "system",
	Address: "",
}, {
	Network: "udp",
	Address: "8.8.4.4:53",
}, {
	Network: "udp",
	Address: "1.1.1.1:53",
}}

func (mx *Measurer) runAsync(ctx context.Context, sess model.ExperimentSession,
	URL string, th *model.Service, out chan<- *model.ExperimentAsyncTestKeys) {
	defer close(out)
	helper := &measurerMeasureURLHelper{
		Clnt:   sess.DefaultHTTPClient(),
		Logger: sess.Logger(),
		THURL:  th.Address,
	}
	mmx := &measurex.Measurer{
		Begin:            time.Now(),
		HTTPClient:       sess.DefaultHTTPClient(),
		MeasureURLHelper: helper,
		Logger:           sess.Logger(),
		Resolvers:        measurerResolvers,
		TLSHandshaker:    netxlite.NewTLSHandshakerDefault(sess.Logger()),
	}
	cookies := measurex.NewCookieJar()
	in := mmx.MeasureURLAndFollowRedirections(
		ctx, URL, measurex.NewHTTPRequestHeaderForMeasuring(), cookies)
	for m := range in {
		out <- &model.ExperimentAsyncTestKeys{
			Extensions: map[string]int64{
				archival.ExtHTTP.Name:         archival.ExtHTTP.V,
				archival.ExtDNS.Name:          archival.ExtDNS.V,
				archival.ExtNetevents.Name:    archival.ExtNetevents.V,
				archival.ExtTCPConnect.Name:   archival.ExtTCPConnect.V,
				archival.ExtTLSHandshake.Name: archival.ExtTLSHandshake.V,
			},
			Input:              model.MeasurementTarget(m.URL),
			MeasurementRuntime: m.TotalRuntime.Seconds(),
			TestKeys:           &TestKeys{URLMeasurement: m},
		}
	}
}

// measurerMeasureURLHelper injects the TH into the normal
// URL measurement flow implemented by measurex.
type measurerMeasureURLHelper struct {
	// Clnt is the MANDATORY client to use
	Clnt measurex.HTTPClient

	// Logger is the MANDATORY Logger to use
	Logger model.Logger

	// THURL is the MANDATORY TH URL.
	THURL string
}

func (mth *measurerMeasureURLHelper) LookupExtraHTTPEndpoints(
	ctx context.Context, URL *url.URL, headers http.Header,
	curEndpoints ...*measurex.HTTPEndpoint) (
	[]*measurex.HTTPEndpoint, interface{}, error) {
	cc := &THClientCall{
		Endpoints:  measurex.HTTPEndpointsToEndpoints(curEndpoints),
		HTTPClient: mth.Clnt,
		Header:     headers,
		THURL:      mth.THURL,
		TargetURL:  URL.String(),
	}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	ol := measurex.NewOperationLogger(
		mth.Logger, "THClientCall %s", URL.String())
	resp, err := cc.Call(ctx)
	ol.Stop(err)
	if err != nil {
		return nil, resp, err
	}
	var out []*measurex.HTTPEndpoint
	for _, epnt := range resp.Endpoints {
		out = append(out, &measurex.HTTPEndpoint{
			Domain:  URL.Hostname(),
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     URL.Hostname(),
			ALPN:    measurex.ALPNForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  headers,
		})
	}
	return out, resp, nil
}

// Run implements ExperimentMeasurer.Run.
func (mx *Measurer) Run(ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks) error {
	return errors.New("sync run is not implemented")
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	Accessible bool   `json:"accessible"`
	Blocking   string `json:"blocking"`
	IsAnomaly  bool   `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (mx *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{}
	return sk, nil
}
