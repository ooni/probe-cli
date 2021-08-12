// Package nwcth implements the new web connectivity test helper.
//
// See https://github.com/ooni/spec/blob/master/backends/th-007-nwcth.md
package nwcth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// newfailure is a convenience shortcut to save typing
var newfailure = archival.NewFailure

// maxAcceptableBody is _at the same time_ the maximum acceptable body for incoming
// API requests and the maximum acceptable body when fetching arbitrary URLs. See
// https://github.com/ooni/probe/issues/1727 for statistics regarding the test lists
// including the empirical CDF of the body size for test lists URLs.
const maxAcceptableBody = 1 << 24

// Handler implements the Web Connectivity test helper HTTP API.
type Handler struct{}

// ServeHTTP implements http.Handler.ServeHTTP.
func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s", version.Version, version.Version,
	))
	if req.Method != "POST" {
		w.WriteHeader(400)
		return
	}
	reader := &io.LimitedReader{R: req.Body, N: maxAcceptableBody}
	data, err := iox.ReadAllContext(req.Context(), reader)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var creq ControlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		w.WriteHeader(400)
		return
	}
	cresp, err := Measure(req.Context(), &creq)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	// We assume that the following call cannot fail because it's a
	// clearly serializable data structure.
	data, err = json.Marshal(cresp)
	runtimex.PanicOnError(err, "json.Marshal failed")
	w.Header().Add("Content-Type", "application/json")
	w.Write(data)
}
