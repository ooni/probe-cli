package ooapi

//
// Web Connectivity Test Helper (TH).
//

import (
	"encoding/json"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewDescriptorTH creates a new [httpapi.Descriptor] describing how
// to issue an HTTP call to the Web Connectivity Test Helper (TH).
func NewDescriptorTH(
	creq *model.THRequest,
) *httpapi.Descriptor[*model.THRequest, *model.THResponse] {
	rawRequest, err := json.Marshal(creq)
	runtimex.PanicOnError(err, "json.Marshal failed unexpectedly")
	return &httpapi.Descriptor[*model.THRequest, *model.THResponse]{
		Accept:             httpapi.ApplicationJSON,
		AcceptEncodingGzip: false,
		Authorization:      "",
		ContentType:        httpapi.ApplicationJSON,
		LogBody:            true,
		MaxBodySize:        0,
		Method:             http.MethodPost,
		Request: &httpapi.RequestDescriptor[*model.THRequest]{
			Body: rawRequest,
		},
		Response: &httpapi.JSONResponseDescriptor[model.THResponse]{},
		Timeout:  0,
		URLPath:  "/",
		URLQuery: nil,
	}
}
