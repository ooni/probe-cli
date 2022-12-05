package ooapi

import (
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestHelpersAPISpec is the spec of the test-helpers API.
type TestHelpersAPISpec struct{}

var _ httpapi.TypedSpec[map[string][]model.OOAPIService] = &TestHelpersAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *TestHelpersAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   "", // not needed
		LogBody:       true,
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil, // none
		Timeout:       20 * time.Second,
		URLPath:       "/api/v1/test-helpers",
		URLQuery:      nil, // none
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *TestHelpersAPISpec) ZeroResponse() map[string][]model.OOAPIService {
	return make(map[string][]model.OOAPIService)
}
