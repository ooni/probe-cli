// Package oonirun implements OONI Run v1 and v2.
package oonirun

//
// Top-level entry point
//

import (
	"context"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Config contains config for OONI Run. You MUST fill all the fields that
// are marked as MANDATORY, or the Config would cause crashes.
type Config struct {
	// AcceptChanges is OPTIONAL and tells this library that the user is
	// okay with running a new or modified OONI Run link without previously
	// reviewing what it contains or what has changed.
	AcceptChanges bool

	// Annotations contains OPTIONAL Annotations for the experiment.
	Annotations map[string]string

	// KVStore is the MANDATORY key-value store to use to keep track of
	// OONI Run links and know when they are new or modified.
	KVStore model.KeyValueStore

	// MaxRuntime is the OPTIONAL maximum runtime in seconds.
	MaxRuntime int64

	// NoCollector OPTIONALLY indicates we should not be using any collector.
	NoCollector bool

	// NoJSON OPTIONALLY indicates we don't want to save measurements to a JSON file.
	NoJSON bool

	// Random OPTIONALLY indicates we should randomize inputs.
	Random bool

	// ReportFile is the MANDATORY file in which to save reports, which is only
	// used when noJSON is set to false.
	ReportFile string

	// Session is the MANDATORY Session to use.
	Session Session
}

// Session is the definition of Session used by this package.
type Session interface {
	// A Session is also an InputLoaderSession.
	engine.InputLoaderSession

	// A Session is also a SubmitterSession.
	engine.SubmitterSession

	// DefaultHTTPClient returns the session's default HTTPClient.
	DefaultHTTPClient() model.HTTPClient

	// Logger returns the logger used by this Session.
	Logger() model.Logger

	// NewExperimentBuilder creates a new engine.ExperimentBuilder.
	NewExperimentBuilder(name string) (engine.ExperimentBuilder, error)
}

// Measure performs the measurement indicated by the given OONI Run link.
func Measure(ctx context.Context, config *Config, URL string) error {
	config.Session.Logger().Infof("oonirun: loading measurement list from %s", URL)
	switch {
	case strings.HasPrefix(URL, "https://run.ooni.io/nettest"):
		return v1Measure(ctx, config, URL)
	case strings.HasPrefix(URL, "ooni://nettest"):
		return v1Measure(ctx, config, URL)
	default:
		return v2MeasureHTTPS(ctx, config, URL)
	}
}
