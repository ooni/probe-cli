package webconnectivity

//
// Measurer
//

import (
	"context"
	"errors"
	"net/http/cookiejar"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/publicsuffix"
)

// Measurer for the web_connectivity experiment.
type Measurer struct {
	// Contains the experiment's config.
	Config *Config
}

// NewExperimentMeasurer creates a new model.ExperimentMeasurer.
func NewExperimentMeasurer(config *Config) model.ExperimentMeasurer {
	return &Measurer{
		Config: config,
	}
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string {
	return "web_connectivity"
}

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string {
	return "0.5.19"
}

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	// Reminder: when this function returns an error, the measurement result
	// WILL NOT be submitted to the OONI backend. You SHOULD only return an error
	// for fundamental errors (e.g., the input is invalid or missing).
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// make sure we have a cancellable context such that we can stop any
	// goroutine running in the background (e.g., priority.go's ones)
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// honour InputOrQueryBackend
	input := measurement.Input
	if input == "" {
		return errors.New("no input provided")
	}

	// convert the input string to a URL
	inputParser := &InputParser{
		AcceptedSchemes: []string{
			"http",
			"https",
		},
		AllowEndpoints: false,
		DefaultScheme:  "",
	}
	URL, err := inputParser.Parse(string(measurement.Input))
	if err != nil {
		return err
	}

	// initialize the experiment's test keys
	tk := NewTestKeys()
	measurement.TestKeys = tk

	// create variables required to run parallel tasks
	idGenerator := &atomicx.Int64{}
	wg := &sync.WaitGroup{}

	// create cookiejar
	jar, err := cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	})
	if err != nil {
		return err
	}

	// obtain the test helper's address
	testhelpers, _ := sess.GetTestHelpersByName("web-connectivity")
	if len(testhelpers) < 1 {
		sess.Logger().Warnf("continuing without a valid TH address")
		tk.SetControlFailure(webconnectivity.ErrNoAvailableTestHelpers)
	}

	registerExtensions(measurement)

	// start background tasks
	resos := &DNSResolvers{
		DNSCache:     NewDNSCache(),
		Domain:       URL.Hostname(),
		IDGenerator:  idGenerator,
		Logger:       sess.Logger(),
		NumRedirects: NewNumRedirects(5),
		TestKeys:     tk,
		URL:          URL,
		ZeroTime:     measurement.MeasurementStartTimeSaved,
		WaitGroup:    wg,
		CookieJar:    jar,
		Referer:      "",
		Session:      sess,
		TestHelpers:  testhelpers,
		UDPAddress:   "",
	}
	resos.Start(ctx)

	// wait for background tasks to join
	wg.Wait()

	// If the context passed to us has been cancelled, we cannot
	// trust this experiment's results to be okay.
	if err := ctx.Err(); err != nil {
		return err
	}

	// perform any deferred computation on the test keys
	tk.Finalize(sess.Logger())

	// set the test helper we used
	// TODO(bassosimone): it may be more informative to know about all the
	// test helpers we _tried_ to use, however the data format does not have
	// support for that as far as I can tell...
	if th := tk.getTestHelper(); th != nil {
		measurement.TestHelpers = map[string]interface{}{
			"backend": th,
		}
	}

	// return whether there was a fundamental failure, which would prevent
	// the measurement from being submitted to the OONI collector.
	return tk.fundamentalFailure
}

// registerExtensions registers the extensions used by this
// experiment into the given measurement.
func registerExtensions(m *model.Measurement) {
	model.ArchivalExtHTTP.AddTo(m)
	model.ArchivalExtDNS.AddTo(m)
	model.ArchivalExtNetevents.AddTo(m)
	model.ArchivalExtTCPConnect.AddTo(m)
	model.ArchivalExtTLSHandshake.AddTo(m)
	model.ArchivalExtTunnel.AddTo(m)
}
