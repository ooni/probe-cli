package ooapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// RegisterAPISpec is the spec for the register API.
type RegisterAPISpec struct {
	Request *model.OOAPIRegisterRequest
}

var _ httpapi.TypedSpec[model.OOAPIRegisterResponse] = &RegisterAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *RegisterAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "",
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // it's fine to log this body
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   data,
		Timeout:       30 * time.Second,
		URLPath:       "/api/v1/register",
		URLQuery:      nil,
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *RegisterAPISpec) ZeroResponse() model.OOAPIRegisterResponse {
	return model.OOAPIRegisterResponse{}
}
