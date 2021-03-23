package webconnectivity

import (
	"context"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// ConnectsConfig contains the config for Connects
type ConnectsConfig struct {
	Begin         time.Time
	Session       model.ExperimentSession
	TargetURL     *url.URL
	URLGetterURLs []string
}

// TODO(bassosimone): we should normalize the timings

// ConnectsResult contains the results of Connects
type ConnectsResult struct {
	AllKeys   []urlgetter.TestKeys
	Successes int
	Total     int
}

// Connects performs 0..N connects (either using TCP or TLS) to
// check whether the resolved endpoints are reachable.
func Connects(ctx context.Context, config ConnectsConfig) (out ConnectsResult) {
	out.AllKeys = []urlgetter.TestKeys{}
	multi := urlgetter.Multi{Begin: config.Begin, Session: config.Session}
	inputs := []urlgetter.MultiInput{}
	for _, url := range config.URLGetterURLs {
		inputs = append(inputs, urlgetter.MultiInput{
			Config: urlgetter.Config{
				TLSServerName: config.TargetURL.Hostname(),
			},
			Target: url,
		})
	}
	outputs := multi.Collect(ctx, inputs, "check", ConnectsNoCallbacks{})
	for multiout := range outputs {
		out.AllKeys = append(out.AllKeys, multiout.TestKeys)
		for _, entry := range multiout.TestKeys.TCPConnect {
			if entry.Status.Success {
				out.Successes++
			}
			out.Total++
		}
	}
	return
}

// ConnectsNoCallbacks suppresses the callbacks
type ConnectsNoCallbacks struct{}

// OnProgress implements ExperimentCallbacks.OnProgress
func (ConnectsNoCallbacks) OnProgress(percentage float64, message string) {}
