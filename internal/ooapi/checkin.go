package ooapi

//
// CheckIn API
//

import (
	"encoding/json"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// NewDescriptorCheckIn creates a new [httpapi.Descriptor] describing how
// to issue an HTTP call to the CheckIn API.
func NewDescriptorCheckIn(
	config *model.OOAPICheckInConfig,
) *httpapi.Descriptor[*model.OOAPICheckInConfig, *model.OOAPICheckInResult] {
	rawRequest, err := json.Marshal(config)
	runtimex.PanicOnError(err, "json.Marshal failed unexpectedly")
	return &httpapi.Descriptor[*model.OOAPICheckInConfig, *model.OOAPICheckInResult]{
		Accept:             httpapi.ApplicationJSON,
		AcceptEncodingGzip: true, // we want a small response
		Authorization:      "",
		ContentType:        httpapi.ApplicationJSON,
		LogBody:            true, // we don't want to log psiphon config
		MaxBodySize:        0,
		Method:             http.MethodPost,
		Request: &httpapi.RequestDescriptor[*model.OOAPICheckInConfig]{
			Body: rawRequest,
		},
		Response: &httpapi.JSONResponseDescriptor[model.OOAPICheckInResult]{},
		Timeout:  0,
		URLPath:  "/api/v1/check-in",
		URLQuery: nil,
	}
}
