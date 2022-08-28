package main

//
// HTTP handler
//

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// handler implements the Web Connectivity test helper HTTP API.
type handler struct {
	// BaseLogger is the MANDATORY logger to use.
	BaseLogger model.Logger

	// Indexer is the MANDATORY atomic integer used to assign an index to requests.
	Indexer *atomicx.Int64

	// MaxAcceptableBody is the MANDATORY maximum acceptable response body.
	MaxAcceptableBody int64

	// NewClient is the MANDATORY factory to create a new HTTPClient.
	NewClient func(model.Logger) model.HTTPClient

	// NewDialer is the MANDATORY factory to create a new Dialer.
	NewDialer func(model.Logger) model.Dialer

	// NewResolver is the MANDATORY factory for creating a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// NewTLSHandshaker is the MANDATORY factory for creating a new TLS handshaker.
	NewTLSHandshaker func(model.Logger) model.TLSHandshaker
}

var _ http.Handler = &handler{}

// ServeHTTP implements http.Handler.ServeHTTP.
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	metricRequestsInflight.Inc()
	defer metricRequestsInflight.Dec()
	metricRequestsTotal.Inc()
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s", version.Version, version.Version,
	))
	if req.Method != "POST" {
		metricRequestsByStatusCode.WithLabelValues("400").Inc()
		w.WriteHeader(400)
		return
	}
	reader := &io.LimitedReader{R: req.Body, N: h.MaxAcceptableBody}
	data, err := netxlite.ReadAllContext(req.Context(), reader)
	if err != nil {
		metricRequestsByStatusCode.WithLabelValues("400").Inc()
		w.WriteHeader(400)
		return
	}
	var creq ctrlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		metricRequestsByStatusCode.WithLabelValues("400").Inc()
		w.WriteHeader(400)
		return
	}
	started := time.Now()
	cresp, err := measure(req.Context(), h, &creq)
	elapsed := time.Since(started)
	metricMeasurementTime.Observe(float64(elapsed.Seconds()))
	if err != nil {
		metricRequestsByStatusCode.WithLabelValues("400").Inc()
		w.WriteHeader(400)
		return
	}
	metricRequestsByStatusCode.WithLabelValues("200").Inc()
	// We assume that the following call cannot fail because it's a
	// clearly-serializable data structure.
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
