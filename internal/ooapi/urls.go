package ooapi

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// URLListAPISpec is the spec for the url-list API
type URLListAPISpec struct {
	Config *model.OOAPIURLListConfig
}

var _ httpapi.TypedSpec[model.OOAPIURLListResult] = &URLListAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *URLListAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	query := url.Values{}
	if spec.Config.CountryCode != "" {
		query.Set("country_code", spec.Config.CountryCode)
	}
	if spec.Config.Limit > 0 {
		query.Set("limit", fmt.Sprintf("%d", spec.Config.Limit))
	}
	if len(spec.Config.Categories) > 0 {
		// Note: ooapi (the unused package in v3.14.0 that implemented automatic API
		// generation) used `category_code` (singular) here, but that's wrong. The plural
		// name is the correct name as I've just verified -- 2022-11-30.
		query.Set("category_codes", strings.Join(spec.Config.Categories, ","))
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // okay to log
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil, // none
		Timeout:       60 * time.Second,
		URLPath:       "/api/v1/test-list/urls",
		URLQuery:      query,
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *URLListAPISpec) ZeroResponse() model.OOAPIURLListResult {
	return model.OOAPIURLListResult{}
}
