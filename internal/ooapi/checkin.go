package ooapi

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CheckInAPISpec is the spec of the check-in API.
type CheckInAPISpec struct {
	Request *model.OOAPICheckInConfig
}

// TODO(bassosimone): I wonder whether ZeroResponse should belong to
// the TypedSpec[T] or instead it should belong to some other interface:
// so far the whole API looks a bit clumsy.

var _ httpapi.TypedSpec[model.OOAPICheckInResult] = &CheckInAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *CheckInAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       false, // contains psiphon data
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   data,
		Timeout:       60 * time.Second,
		URLPath:       "/api/v1/check-in",
		URLQuery:      nil, // none
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *CheckInAPISpec) ZeroResponse() model.OOAPICheckInResult {
	return model.OOAPICheckInResult{}
}
