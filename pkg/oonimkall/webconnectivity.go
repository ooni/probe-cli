package oonimkall

import (
	"context"
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// WebConnectivityConfig contains settings for WebConnectivity.
//
// Deprecated: this code was just an experiment that we never ended up using.
type WebConnectivityConfig struct {
	// Callbacks contains the experiment callbacks. This field is
	// optional. Leave it empty and we'll use a default set of
	// callbacks that use the session logger.
	Callbacks ExperimentCallbacks

	// Input contains the URL to measure. This field must be set
	// by the user, otherwise the experiment fails.
	Input string
}

// WebConnectivityResults contains the results of WebConnectivity.
//
// Deprecated: this code was just an experiment that we never ended up using.
type WebConnectivityResults struct {
	// KibiBytesReceived contains the KiB received.
	KibiBytesReceived float64

	// KibiBytesSent contains the KiB sent.
	KibiBytesSent float64

	// Measurement contains the resulting measurement.
	Measurement string
}

// webConnectivityRunner is the type that runs
// the WebConnectivity experiment.
type webConnectivityRunner struct {
	sess experimentSession
}

// run runs the WebConnectivity experiment to completion. Both arguments
// must be correctly initialized. The return value is either a valid
// results with a nil error, or nil results with an error.
func (r *webConnectivityRunner) run(ctx context.Context, config *WebConnectivityConfig) (*WebConnectivityResults, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // helps with testing
	default:
		// fallthrough
	}
	// TODO(bassosimone): I suspect most of the code for running
	// experiments is going to be quite redundant. Autogen?
	defer r.sess.unlock()
	r.sess.lock()
	if err := r.sess.maybeLookupBackends(ctx); err != nil {
		return nil, err
	}
	if err := r.sess.maybeLookupLocation(ctx); err != nil {
		return nil, err
	}
	builder, err := r.sess.newExperimentBuilder("web_connectivity")
	if err != nil {
		return nil, err
	}
	if config.Callbacks != nil {
		builder.setCallbacks(config.Callbacks)
	}
	exp := builder.newExperiment()
	target := model.NewOOAPIURLInfoWithDefaultCategoryAndCountry(config.Input)
	measurement, err := exp.MeasureWithContext(ctx, target)
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(measurement)
	runtimex.PanicOnError(err, "json.Marshal should not fail here")
	return &WebConnectivityResults{
		KibiBytesReceived: exp.KibiBytesReceived(),
		KibiBytesSent:     exp.KibiBytesSent(),
		Measurement:       string(data),
	}, nil
}

// WebConnectivity runs the WebConnectivity experiment. Both ctx and config
// MUST NOT be nil. Returns either an error or the experiment results.
//
// This function locks the session until it's done. That is, no other operation
// can be performed as long as this function is pending.
//
// This API is currently experimental. We do not promise that we will bump
// the major version number when changing it.
//
// Deprecated: this code was just an experiment that we never ended up using.
func (sess *Session) WebConnectivity(ctx *Context, config *WebConnectivityConfig) (*WebConnectivityResults, error) {
	return (&webConnectivityRunner{sess: sess}).run(ctx.ctx, config)
}
