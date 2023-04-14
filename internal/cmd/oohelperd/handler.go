package main

//
// HTTP handler
//

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// handler is an [http.Handler] implementing the Web
// Connectivity test helper HTTP API.
type handler struct {
	// BaseLogger is the MANDATORY logger to use.
	BaseLogger model.Logger

	// Indexer is the MANDATORY atomic integer used to assign an index to requests.
	Indexer *atomic.Int64

	// MaxAcceptableBody is the MANDATORY maximum acceptable response body.
	MaxAcceptableBody int64

	// Measure is the MANDATORY function that the handler should call
	// for producing a response for a valid incoming request.
	Measure func(ctx context.Context, config *handler, creq *model.THRequest) (*model.THResponse, error)

	// NewDialer is the MANDATORY factory to create a new Dialer.
	NewDialer func(model.Logger) model.Dialer

	// NewHTTPClient is the MANDATORY factory to create a new HTTPClient.
	NewHTTPClient func(model.Logger) model.HTTPClient

	// NewHTTP3Client is the MANDATORY factory to create a new HTTP3Client.
	NewHTTP3Client func(model.Logger) model.HTTPClient

	// NewQUICDialer is the MANDATORY factory to create a new QUICDialer.
	NewQUICDialer func(model.Logger) model.QUICDialer

	// NewResolver is the MANDATORY factory for creating a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// NewTLSHandshaker is the MANDATORY factory for creating a new TLS handshaker.
	NewTLSHandshaker func(model.Logger) model.TLSHandshaker
}

var _ http.Handler = &handler{}

// ServeHTTP implements http.Handler.
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	// track the number of in-flight requests
	metricRequestsInflight.Inc()
	defer metricRequestsInflight.Dec()

	// create and add the Server header
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s",
		version.Version,
		version.Version,
	))

	// we only handle the POST method
	if req.Method != "POST" {
		metricRequestsCount.WithLabelValues("400", "bad_request_method").Inc()
		w.WriteHeader(400)
		return
	}

	// read and parse request body
	reader := io.LimitReader(req.Body, h.MaxAcceptableBody)
	data, err := netxlite.ReadAllContext(req.Context(), reader)
	if err != nil {
		metricRequestsCount.WithLabelValues("400", "request_body_too_large").Inc()
		w.WriteHeader(400)
		return
	}
	var creq ctrlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		metricRequestsCount.WithLabelValues("400", "cannot_unmarshal_request_body").Inc()
		w.WriteHeader(400)
		return
	}

	// measure the given input
	started := time.Now()
	cresp, err := h.Measure(req.Context(), h, &creq)
	elapsed := time.Since(started)

	// track the time required to produce a response
	metricWCTaskDurationSeconds.Observe(float64(elapsed.Seconds()))

	// handle the case of fundamental failure
	if err != nil {
		metricRequestsCount.WithLabelValues("400", "wctask_failed").Inc()
		w.WriteHeader(400)
		return
	}

	// produce successful response.
	//
	// Note: we assume that json.Marshal cannot fail because it's a
	// clearly-serializable data structure.
	metricRequestsCount.WithLabelValues("200", "ok").Inc()
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
