package ooapi

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TorAPISpec is a spec for the tor API.
type TorAPISpec struct {
	// Country contains the MANDATORY country code
	Country string

	// Token is the MANDATORY token to authenticate the call.
	Token string
}

var _ httpapi.TypedSpec[map[string]model.OOAPITorTarget] = &TorAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *TorAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	if spec.Token == "" {
		return nil, ErrEmptyAuthToken
	}
	country := spec.Country
	if country == "" {
		country = "ZZ"
	}
	query := url.Values{}
	query.Add("country_code", country)
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: fmt.Sprintf("Bearer %s", spec.Token),
		ContentType:   "",    // none
		LogBody:       false, // don't want to log tor bridges
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil, // none
		Timeout:       60 * time.Second,
		URLPath:       "/api/v1/test-list/tor-targets",
		URLQuery:      query,
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *TorAPISpec) ZeroResponse() map[string]model.OOAPITorTarget {
	return make(map[string]model.OOAPITorTarget)
}
