package ooapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CollectorOpenReportAPISpec is the API spec for the API
// allowing to open a new report with the collector.
type CollectorOpenReportAPISpec struct {
	Request *model.OOAPIReportTemplate
}

var _ httpapi.TypedSpec[model.OOAPICollectorOpenResponse] = &CollectorOpenReportAPISpec{}

// Descriptor implements httpapi.TypedSpec
func (spec *CollectorOpenReportAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // okay to log
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   data,
		Timeout:       30 * time.Second,
		URLPath:       "/report",
		URLQuery:      nil, // none
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *CollectorOpenReportAPISpec) ZeroResponse() model.OOAPICollectorOpenResponse {
	return model.OOAPICollectorOpenResponse{}
}

// CollectorUpdateReportAPISpec is the spec for the API
// allowing to append a measurement to a report.
type CollectorUpdateReportAPISpec struct {
	ReportID string
	Request  *model.OOAPICollectorUpdateRequest
}

var _ httpapi.TypedSpec[model.OOAPICollectorUpdateResponse] = &CollectorUpdateReportAPISpec{}

// ErrMissingReportID indicates that the reportID is missing.
var ErrMissingReportID = errors.New("missing report ID")

// Descriptor implements httpapi.TypedSpec
func (spec *CollectorUpdateReportAPISpec) Descriptor() (*httpapi.Descriptor, error) {
	if spec.ReportID == "" {
		return nil, ErrMissingReportID
	}
	data, err := json.Marshal(spec.Request)
	if err != nil {
		return nil, err
	}
	desc := &httpapi.Descriptor{
		Accept:        httpapi.ApplicationJSON,
		Authorization: "", // not needed
		ContentType:   httpapi.ApplicationJSON,
		LogBody:       true, // that's fine
		MaxBodySize:   httpapi.DefaultMaxBodySize,
		Method:        http.MethodPost,
		RequestBody:   data,
		Timeout:       120 * time.Second,
		URLPath:       fmt.Sprintf("/report/%s", spec.ReportID),
		URLQuery:      nil, // none
	}
	return desc, nil
}

// ZeroResponse implements httpapi.TypedSpec
func (spec *CollectorUpdateReportAPISpec) ZeroResponse() model.OOAPICollectorUpdateResponse {
	return model.OOAPICollectorUpdateResponse{}
}
