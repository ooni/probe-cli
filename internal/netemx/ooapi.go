package netemx

import (
	"encoding/json"
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// OOAPIHandlerFactory is a [QAEnvHTTPHandlerFactory] that creates [OOAPIHandler] instances.
type OOAPIHandlerFactory struct{}

var _ HTTPHandlerFactory = &OOAPIHandlerFactory{}

// NewHandler implements QAEnvHTTPHandlerFactory.
func (*OOAPIHandlerFactory) NewHandler(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
	return &OOAPIHandler{}
}

// OOAPIHandler is an [http.Handler] implementing the OONI API.
type OOAPIHandler struct{}

var _ http.Handler = &OOAPIHandler{}

// ServeHTTP implements [http.Handler].
func (p *OOAPIHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/api/v1/test-helpers" && r.Method == http.MethodGet:
		p.getApiV1TestHelpers(w, r)

	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

func (p *OOAPIHandler) getApiV1TestHelpers(w http.ResponseWriter, _ *http.Request) {
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
	_, _ = w.Write(runtimex.Try1(json.Marshal(resp)))
}
