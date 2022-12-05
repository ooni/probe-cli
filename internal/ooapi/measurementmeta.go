package ooapi

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// MeasurementMetaAPI spec is the spec for the measurement-meta API.
type MeasurementMetaAPI struct {
	// ReportID is the MANDATORY report ID
	ReportID string

	// Input is the OPTIONAL input
	Input string

	// Full OPTIONALLY indicates whether to retrieve the original measurement
	Full bool
}

var _ httpapi.TypedSpec[model.OOAPIMeasurementMeta] = &MeasurementMetaAPI{}

// Descriptor implements httpapi.TypedSpec
func (spec *MeasurementMetaAPI) Descriptor() (*httpapi.Descriptor, error) {
	query := url.Values{}
	if spec.ReportID == "" {
		return nil, ErrMissingReportID
	}
	query.Add("report_id", spec.ReportID)
	if spec.Input != "" {
		query.Add("input", spec.Input)
	}
	if spec.Full {
		query.Add("full", "true")
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   "", // none
		LogBody:       true,
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodGet,
		RequestBody:   nil,
		Timeout:       60 * time.Second,
		URLPath:       "/api/v1/measurement_meta",
		URLQuery:      query,
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *MeasurementMetaAPI) ZeroResponse() model.OOAPIMeasurementMeta {
	return model.OOAPIMeasurementMeta{}
}
