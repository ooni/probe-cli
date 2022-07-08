package oonirun

//
// OONI Run v1 and v2 links
//

import (
	"context"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// LinkConfig contains config for an OONI Run link. You MUST fill all the fields that
// are marked as MANDATORY, or the LinkConfig would cause crashes.
type LinkConfig struct {
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

// LinkRunner knows how to run an OONI Run v1 or v2 link.
type LinkRunner interface {
	Run(ctx context.Context) error
}

// linkRunner implements LinkRunner.
type linkRunner struct {
	config *LinkConfig
	f      func(ctx context.Context, config *LinkConfig, URL string) error
	url    string
}

// Run implements LinkRunner.Run.
func (lr *linkRunner) Run(ctx context.Context) error {
	return lr.f(ctx, lr.config, lr.url)
}

// NewLinkRunner creates a suitable link runner for the current config
// and the given URL, which is one of the following:
//
// 1. OONI Run v1 link with https scheme (e.g., https://run.ooni.io/nettest?...)
//
// 2. OONI Run v1 link with ooni scheme (e.g., ooni://nettest?...)
//
// 3. arbitrary URL of the OONI Run v2 descriptor.
func NewLinkRunner(c *LinkConfig, URL string) LinkRunner {
	// TODO(bassosimone): add support for v2 deeplinks.
	out := &linkRunner{
		config: c,
		f:      nil,
		url:    URL,
	}
	switch {
	case strings.HasPrefix(URL, "https://run.ooni.io/nettest"):
		out.f = v1Measure
	case strings.HasPrefix(URL, "ooni://nettest"):
		out.f = v1Measure
	default:
		out.f = v2MeasureHTTPS
	}
	return out
}
