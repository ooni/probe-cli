// Package webstepsx contains a websteps implementation
// based on the internal/measurex package.
//
// This implementation does not follow any existing spec
// rather we are modeling the spec on this one.
package webstepsx

import (
	"context"
	"errors"
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
	*measurex.ArchivalURLMeasurement
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
	out := make(chan *model.ExperimentAsyncTestKeys)
	go mx.runAsync(ctx, sess, input, testhelper, out)
	return out, nil
}

func (mx *Measurer) runAsync(ctx context.Context, sess model.ExperimentSession,
	URL string, th *model.Service, out chan<- *model.ExperimentAsyncTestKeys) {
	defer close(out)
	begin := time.Now()
	db := measurex.NewDB(begin)
	mmx := &measurex.Measurer{
		DB:            db,
		HTTPClient:    sess.DefaultHTTPClient(),
		Logger:        sess.Logger(),
		Origin:        measurex.OriginProbe,
		TLSHandshaker: netxlite.NewTLSHandshakerStdlib(sess.Logger()),
	}
	mmx.RegisterUDPResolvers("8.8.4.4:53", "8.8.8.8:53", "1.1.1.1:53", "1.0.0.1:53")
	mmx.RegisterWCTH(th.Address)
	cookies := measurex.NewCookieJar()
	in := mmx.MeasureHTTPURLAndFollowRedirections(ctx, URL, cookies)
	for m := range in {
		out <- &model.ExperimentAsyncTestKeys{
			MeasurementRuntime: time.Since(begin).Seconds(),
			TestKeys: &TestKeys{
				measurex.NewArchivalURLMeasurement(m),
			},
			Extensions: map[string]int64{
				archival.ExtHTTP.Name:         archival.ExtHTTP.V,
				archival.ExtDNS.Name:          archival.ExtDNS.V,
				archival.ExtNetevents.Name:    archival.ExtNetevents.V,
				archival.ExtTCPConnect.Name:   archival.ExtTCPConnect.V,
				archival.ExtTLSHandshake.Name: archival.ExtTLSHandshake.V,
			},
		}
	}
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
