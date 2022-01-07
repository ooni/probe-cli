// Package tlstool contains a TLS tool that we are currently using
// for running quick and dirty experiments. This tool will change
// without notice and may be removed without notice.
//
// Caveats
//
// In particular, this experiment MAY panic when passed incorrect
// input. This is acceptable because this is not production ready code.
package tlstool

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlstool/internal"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	testName    = "tlstool"
	testVersion = "0.1.0"
)

// Config contains the experiment configuration.
type Config struct {
	Delay int64  `ooni:"Milliseconds to wait between writes"`
	SNI   string `ooni:"Force using the specified SNI"`
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Experiment map[string]*ExperimentKeys `json:"experiment"`
}

// ExperimentKeys contains the specific experiment results.
type ExperimentKeys struct {
	Failure *string `json:"failure"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

type method struct {
	name      string
	newDialer func(internal.DialerConfig) internal.Dialer
}

var allMethods = []method{{
	name:      "vanilla",
	newDialer: internal.NewVanillaDialer,
}, {
	name:      "snisplit",
	newDialer: internal.NewSNISplitterDialer,
}, {
	name:      "random",
	newDialer: internal.NewRandomSplitterDialer,
}, {
	name:      "thrice",
	newDialer: internal.NewThriceSplitterDialer,
}}

// Run implements ExperimentMeasurer.Run.
func (m Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	// TODO(bassosimone): wondering whether this experiment should
	// actually be merged with sniblocking instead?
	tk := new(TestKeys)
	tk.Experiment = make(map[string]*ExperimentKeys)
	measurement.TestKeys = tk
	address := string(measurement.Input)
	for idx, meth := range allMethods {
		// TODO(bassosimone): here we actually want to use urlgetter
		// if possible and collect standard test keys.
		err := m.run(ctx, runConfig{
			address:   address,
			logger:    sess.Logger(),
			newDialer: meth.newDialer,
		})
		percent := float64(idx) / float64(len(allMethods))
		callbacks.OnProgress(percent, fmt.Sprintf("%s: %+v", meth.name, err))
		tk.Experiment[meth.name] = &ExperimentKeys{
			Failure: archival.NewFailure(err),
		}
	}
	return nil // return nil so we always submit the measurement
}

func (m Measurer) newDialer(logger model.Logger) netx.Dialer {
	// TODO(bassosimone): this is a resolver that should hopefully work
	// in many places. Maybe allow to configure it?
	resolver, err := netx.NewDNSClientWithOverrides(netx.Config{Logger: logger},
		"https://cloudflare.com/dns-query", "dns.cloudflare.com", "", "")
	runtimex.PanicOnError(err, "cannot initialize resolver")
	return netx.NewDialer(netx.Config{FullResolver: resolver, Logger: logger})
}

type runConfig struct {
	address   string
	logger    model.Logger
	newDialer func(internal.DialerConfig) internal.Dialer
}

func (m Measurer) run(ctx context.Context, config runConfig) error {
	dialer := config.newDialer(internal.DialerConfig{
		Dialer: m.newDialer(config.logger),
		Delay:  time.Duration(m.config.Delay) * time.Millisecond,
		SNI:    m.pattern(config.address),
	})
	tdialer := netx.NewTLSDialer(netx.Config{
		Dialer:    dialer,
		Logger:    config.logger,
		TLSConfig: m.tlsConfig(),
	})
	conn, err := tdialer.DialTLSContext(ctx, "tcp", config.address)
	if err != nil {
		return err
	}
	conn.Close()
	return nil
}

func (m Measurer) tlsConfig() *tls.Config {
	if m.config.SNI != "" {
		return &tls.Config{ServerName: m.config.SNI}
	}
	return nil
}

func (m Measurer) pattern(address string) string {
	if m.config.SNI != "" {
		return m.config.SNI
	}
	addr, _, err := net.SplitHostPort(address)
	// TODO(bassosimone): replace this panic with proper error checking.
	runtimex.PanicOnError(err, "cannot split address")
	return addr
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return SummaryKeys{IsAnomaly: false}, nil
}
