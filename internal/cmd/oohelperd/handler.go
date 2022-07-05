package main

//
// HTTP handler
//

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// handler implements the Web Connectivity test helper HTTP API.
type handler struct {
	// MaxAcceptableBody is the MANDATORY maximum acceptable response body.
	MaxAcceptableBody int64

	// NewClient is the MANDATORY factory to create a new HTTPClient.
	NewClient func() model.HTTPClient

	// NewDialer is the MANDATORY factory to create a new Dialer.
	NewDialer func() model.Dialer

	// NewResolver is the MANDATORY factory for creating a new resolver.
	NewResolver func() model.Resolver
}

var _ http.Handler = &handler{}

// ServeHTTP implements http.Handler.ServeHTTP.
func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s", version.Version, version.Version,
	))
	if req.Method != "POST" {
		w.WriteHeader(400)
		return
	}
	reader := &io.LimitedReader{R: req.Body, N: h.MaxAcceptableBody}
	data, err := netxlite.ReadAllContext(req.Context(), reader)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var creq ctrlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		w.WriteHeader(400)
		return
	}
	cresp, err := measure(req.Context(), h, &creq)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	// We assume that the following call cannot fail because it's a
	// clearly-serializable data structure.
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
