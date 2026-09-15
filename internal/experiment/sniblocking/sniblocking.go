// Package sniblocking contains the SNI blocking network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-024-sni-blocking.md.
package sniblocking

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "sni_blocking"
	testVersion = "0.3.1"
)

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType
)

// Config contains the experiment config.
type Config struct {
	// ControlSNI is the SNI to be used for the control.
	ControlSNI string `json:"control_sni,omitempty"`

	// TestHelperAddress is the address of the test helper.
	TestHelperAddress string `json:"test_helper_address,omitempty"`
}

// Subresult contains the keys of a single measurement
// that targets either the target or the control.
type Subresult struct {
	urlgetter.TestKeys
	Cached    bool   `json:"-"`
	SNI       string `json:"sni"`
	THAddress string `json:"th_address"`
}

// TestKeys contains sniblocking test keys.
type TestKeys struct {
	Control Subresult `json:"control"`
	Result  string    `json:"result"`
	Target  Subresult `json:"target"`
}

const (
	classAnomalyTestHelperUnreachable   = "anomaly.test_helper_unreachable"
	classAnomalyTimeout                 = "anomaly.timeout"
	classAnomalyUnexpectedFailure       = "anomaly.unexpected_failure"
	classInterferenceClosed             = "interference.closed"
	classInterferenceInvalidCertificate = "interference.invalid_certificate"
	classInterferenceReset              = "interference.reset"
	classInterferenceUnknownAuthority   = "interference.unknown_authority"
	classSuccessGotServerHello          = "success.got_server_hello"
)

func (tk *TestKeys) classify() string {
	if tk.Target.Failure == nil {
		return classSuccessGotServerHello
	}
	switch *tk.Target.Failure {
	case netxlite.FailureConnectionRefused:
		return classAnomalyTestHelperUnreachable
	case netxlite.FailureConnectionReset:
		return classInterferenceReset
	case netxlite.FailureDNSNXDOMAINError, netxlite.FailureAndroidDNSCacheNoData:
		return classAnomalyTestHelperUnreachable
	case netxlite.FailureEOFError:
		return classInterferenceClosed
	case netxlite.FailureGenericTimeoutError:
		if tk.Control.Failure != nil {
			return classAnomalyTestHelperUnreachable
		}
		return classAnomalyTimeout
	case netxlite.FailureSSLInvalidCertificate:
		return classInterferenceInvalidCertificate
	case netxlite.FailureSSLInvalidHostname:
		return classSuccessGotServerHello
	case netxlite.FailureSSLUnknownAuthority:
		return classInterferenceUnknownAuthority
	}
	return classAnomalyUnexpectedFailure
}

// Measurer performs the measurement.
type Measurer struct {
	cache map[string]Subresult
	mu    sync.Mutex
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func (m *Measurer) measureone(
	ctx context.Context,
	sess model.ExperimentSession,
	beginning time.Time,
	sni string,
	thaddr string,
) Subresult {
	// slightly delay the measurement
	gen := rand.New(rand.NewSource(time.Now().UnixNano())) // #nosec G404 -- not really important
	sleeptime := time.Duration(gen.Intn(250)) * time.Millisecond
	select {
	case <-time.After(sleeptime):
	case <-ctx.Done():
		s := netxlite.FailureInterrupted
		failedop := netxlite.TopLevelOperation
		return Subresult{
			TestKeys: urlgetter.TestKeys{
				FailedOperation: &failedop,
				Failure:         &s,
			},
			THAddress: thaddr,
			SNI:       sni,
		}
	}
	// perform the measurement
	g := urlgetter.Getter{
		Begin:   beginning,
		Config:  urlgetter.Config{TLSServerName: sni},
		Session: sess,
		Target:  fmt.Sprintf("tlshandshake://%s", thaddr),
	}
	// Ignoring the error because g.Get() sets the tk.Failure field
	// to be the OONI equivalent of the error that occurred.
	tk, _ := g.Get(ctx)
	// assemble and publish the results
	smk := Subresult{
		SNI:       sni,
		THAddress: thaddr,
		TestKeys:  tk,
	}
	return smk
}

func (m *Measurer) measureonewithcache(
	ctx context.Context,
	output chan<- Subresult,
	sess model.ExperimentSession,
	beginning time.Time,
	sni string,
	thaddr string,
) {
	cachekey := sni + thaddr
	m.mu.Lock()
	smk, okay := m.cache[cachekey]
	m.mu.Unlock()
	if okay {
		output <- smk
		return
	}
	smk = m.measureone(ctx, sess, beginning, sni, thaddr)
	output <- smk
	smk.Cached = true
	m.mu.Lock()
	m.cache[cachekey] = smk
	m.mu.Unlock()
}

func (m *Measurer) startall(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, inputs []string, testHelperAddress string,
) <-chan Subresult {
	outputs := make(chan Subresult, len(inputs))
	for _, input := range inputs {
		go m.measureonewithcache(
			ctx, outputs, sess,
			measurement.MeasurementStartTimeSaved,
			input, testHelperAddress,
		)
	}
	return outputs
}

func processall(
	outputs <-chan Subresult,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
	inputs []string,
	sess model.ExperimentSession,
	controlSNI string,
) *TestKeys {
	var (
		current  int
		testkeys = new(TestKeys)
	)
	for smk := range outputs {
		if smk.SNI == controlSNI {
			testkeys.Control = smk
		} else if smk.SNI == string(measurement.Input) {
			testkeys.Target = smk
		} else {
			panic("unexpected smk.SNI")
		}
		current++
		sess.Logger().Debugf(
			"sni_blocking: %s: %s [cached: %+v]", smk.SNI,
			asString(smk.Failure), smk.Cached)
		if current >= len(inputs) {
			break
		}
	}
	testkeys.Result = testkeys.classify()
	sess.Logger().Infof("sni_blocking: result: %s", testkeys.Result)
	return testkeys
}

// maybeURLToSNI handles the case where the input is from the test-lists
// and hence every input is a URL rather than a domain.
func maybeURLToSNI(input model.MeasurementInput) (model.MeasurementInput, error) {
	parsed, err := url.Parse(string(input))
	if err != nil {
		return "", err
	}
	if parsed.Path == string(input) {
		return input, nil
	}
	return model.MeasurementInput(parsed.Hostname()), nil
}

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// obtain the richer-input target
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}
	config, input := target.Config, target.URL

	m.mu.Lock()
	if m.cache == nil {
		m.cache = make(map[string]Subresult)
	}
	m.mu.Unlock()
	if config.ControlSNI == "" {
		config.ControlSNI = "example.org"
	}
	if input == "" {
		return errors.New("experiment requires measurement.Input")
	}
	if config.TestHelperAddress == "" {
		config.TestHelperAddress = net.JoinHostPort(
			config.ControlSNI, "443",
		)
	}
	urlgetter.RegisterExtensions(measurement)
	// TODO(bassosimone): if the user has configured DoT or DoH, here we
	// probably want to perform the name resolution before the measurements
	// or to make sure that the classify logic is robust to that.
	//
	// See https://github.com/ooni/probe-engine/issues/392.
	maybeParsed, err := maybeURLToSNI(model.MeasurementInput(input))
	if err != nil {
		return err
	}
	measurement.Input = maybeParsed
	inputs := []string{config.ControlSNI}
	if string(measurement.Input) != config.ControlSNI {
		inputs = append(inputs, string(measurement.Input))
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second*time.Duration(len(inputs)))
	defer cancel()
	outputs := m.startall(ctx, sess, measurement, inputs, config.TestHelperAddress)
	measurement.TestKeys = processall(
		outputs, measurement, callbacks, inputs, sess, config.ControlSNI,
	)
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}

func asString(failure *string) (result string) {
	result = "success"
	if failure != nil {
		result = *failure
	}
	return
}
