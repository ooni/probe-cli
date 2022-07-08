package oonirun

//
// OONI Run v1 and v2 entry points
//

import (
	"context"
	"strings"

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

// Measure performs the measurement indicated by the given OONI Run link.
func Measure(ctx context.Context, config *Config, URL string) error {
	// TODO(bassosimone): add support for v2 deeplinks.
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
