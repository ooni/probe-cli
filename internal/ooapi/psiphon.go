package ooapi

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
)

// PsiphonConfigAPISpec is the spec for the psiphon-config API.
type PsiphonConfigAPISpec struct {
	// Token is the MANDATORY token to authenticate the call.
	Token string
}

var _ httpapi.SimpleSpec = &PsiphonConfigAPISpec{}

// ErrEmptyAuthToken means the authorization token is empty
var ErrEmptyAuthToken = errors.New("empty auth token")

// Descriptor implements httpapi.SimpleSpec
func (spec *PsiphonConfigAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	if spec.Token == "" {
		return nil, ErrEmptyAuthToken
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: fmt.Sprintf("Bearer %s", spec.Token),
		ContentType:   "",
		LogBody:       false, // don't log psiphon config
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil, // none
		Timeout:       30 * time.Second,
		URLPath:       "/api/v1/test-list/psiphon-config",
		URLQuery:      nil,
	}
	return desc, nil
}
