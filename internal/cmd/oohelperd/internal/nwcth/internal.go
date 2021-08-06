package nwcth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const maxAcceptableBody = 1 << 24

// Handler implements the Web Connectivity test helper HTTP API.
type NWCTHHandler struct{}

func (h NWCTHHandler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Add("Server", fmt.Sprintf(
		"oohelperd/%s ooniprobe-engine/%s", version.Version, version.Version,
	))
	if req.Method != "POST" {
		w.WriteHeader(400)
		return
	}
	if req.Header.Get("content-type") != "application/json" {
		w.WriteHeader(400)
		return
	}
	reader := &io.LimitedReader{R: req.Body, N: maxAcceptableBody}
	data, err := iox.ReadAllContext(req.Context(), reader)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	var creq CtrlRequest
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
