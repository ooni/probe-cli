package ooapi

import (
	"encoding/json"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// THAPISpec specifies the API to communicate with the TH.
type THAPISpec struct {
	Request *model.THRequest
}

var _ httpapi.TypedSpec[model.THResponse] = &THAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *THAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // no need to set authorization
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // completely safe and desirable to log body
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        "POST",
		RequestBody:   data,
		Timeout:       60 * time.Second, // should really be enough
		URLPath:       "/",              // yeah this is the TH's path
		URLQuery:      nil,              // no query
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *THAPISpec) ZeroResponse() model.THResponse {
	return model.THResponse{}
}
