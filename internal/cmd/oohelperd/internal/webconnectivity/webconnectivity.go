package webconnectivity

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// Handler implements the Web Connectivity test helper HTTP API.
type Handler struct {
	Client            *http.Client
	Dialer            netx.Dialer
	MaxAcceptableBody int64
	Resolver          netx.Resolver
}

func (h Handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
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
	var creq CtrlRequest
	if err := json.Unmarshal(data, &creq); err != nil {
		w.WriteHeader(400)
		return
	}
	measureConfig := MeasureConfig(h)
	cresp, err := Measure(req.Context(), measureConfig, &creq)
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
