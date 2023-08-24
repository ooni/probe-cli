package netemx

import (
	"encoding/json"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// OOAPIHandler is an [http.Handler] implementing the OONI API.
type OOAPIHandler struct{}

// ServeHTTP implements [http.Handler].
func (p *OOAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/test-helpers" && r.Method == http.MethodGet:
		p.getApiV1TestHelpers(w, r)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *OOAPIHandler) getApiV1TestHelpers(w http.ResponseWriter, r *http.Request) {
	resp := map[string][]model.OOAPIService{
		"web-connectivity": {
			{
				Address: "https://2.th.ooni.org",
				Type:    "https",
			},
			{
				Address: "https://3.th.ooni.org",
				Type:    "https",
			},
			{
				Address: "https://0.th.ooni.org",
				Type:    "https",
			},
			{
				Address: "https://1.th.ooni.org",
				Type:    "https",
			},
		},
	}
	w.Header().Add("Content-Type", "application/json")
	w.Write(runtimex.Try1(json.Marshal(resp)))
}
