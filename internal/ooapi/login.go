package ooapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// LoginAPISpec is a spec for the login API.
type LoginAPISpec struct {
	Request *model.OOAPILoginAuth
}

var _ httpapi.TypedSpec[model.OOAPILoginCredentials] = &LoginAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *LoginAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // it's fine to log this
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   data,
		Timeout:       30 * time.Second,
		URLPath:       "/api/v1/login",
		URLQuery:      nil, // none
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *LoginAPISpec) ZeroResponse() model.OOAPILoginCredentials {
	return model.OOAPILoginCredentials{}
}
