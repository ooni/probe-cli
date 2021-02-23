package main

import "github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"

// URLPath describes a URLPath.
type URLPath struct {
	IsTemplate bool
	Value      string
	InSwagger  string
}

// Descriptor is an API descriptor.
type Descriptor struct {
	Name          string
	CachePolicy   int
	RequiresLogin bool
	Method        string
	URLPath       URLPath
	Request       interface{}
	Response      interface{}
}

// These are the caching policies.
const (
	// CacheNone indicates we don't use a cache.
	CacheNone = iota

	// CacheFallback indicates we fallback to the cache
	// when there is a failure.
	CacheFallback

	// CacheAlways indicates that we always check the
	// cache before sending a request.
	CacheAlways
)

// Descriptors contains all descriptors.
var Descriptors = []Descriptor{{
	Name:     "CheckReportID",
	Method:   "GET",
	URLPath:  URLPath{Value: "/api/_/check_report_id"},
	Request:  &apimodel.CheckReportIDRequest{},
	Response: &apimodel.CheckReportIDResponse{},
}, {
	Name:        "CheckIn",
	Method:      "POST",
	URLPath:     URLPath{Value: "/api/v1/check-in"},
	Request:     &apimodel.CheckInRequest{},
	Response:    &apimodel.CheckInResponse{},
	CachePolicy: CacheFallback,
}, {
	Name:     "Login",
	Method:   "POST",
	URLPath:  URLPath{Value: "/api/v1/login"},
	Request:  &apimodel.LoginRequest{},
	Response: &apimodel.LoginResponse{},
}, {
	Name:        "MeasurementMeta",
	Method:      "GET",
	URLPath:     URLPath{Value: "/api/v1/measurement_meta"},
	Request:     &apimodel.MeasurementMetaRequest{},
	Response:    &apimodel.MeasurementMetaResponse{},
	CachePolicy: CacheAlways,
}, {
	Name:     "Register",
	Method:   "POST",
	URLPath:  URLPath{Value: "/api/v1/register"},
	Request:  &apimodel.RegisterRequest{},
	Response: &apimodel.RegisterResponse{},
}, {
	Name:        "TestHelpers",
	Method:      "GET",
	URLPath:     URLPath{Value: "/api/v1/test-helpers"},
	Request:     &apimodel.TestHelpersRequest{},
	Response:    apimodel.TestHelpersResponse{},
	CachePolicy: CacheFallback,
}, {
	Name:          "PsiphonConfig",
	RequiresLogin: true,
	Method:        "GET",
	URLPath:       URLPath{Value: "/api/v1/test-list/psiphon-config"},
	Request:       &apimodel.PsiphonConfigRequest{},
	Response:      apimodel.PsiphonConfigResponse{},
}, {
	Name:          "TorTargets",
	RequiresLogin: true,
	Method:        "GET",
	URLPath:       URLPath{Value: "/api/v1/test-list/tor-targets"},
	Request:       &apimodel.TorTargetsRequest{},
	Response:      apimodel.TorTargetsResponse{},
	CachePolicy:   CacheFallback,
}, {
	Name:        "URLs",
	Method:      "GET",
	URLPath:     URLPath{Value: "/api/v1/test-list/urls"},
	Request:     &apimodel.URLsRequest{},
	Response:    &apimodel.URLsResponse{},
	CachePolicy: CacheFallback,
}, {
	Name:     "OpenReport",
	Method:   "POST",
	URLPath:  URLPath{Value: "/report"},
	Request:  &apimodel.OpenReportRequest{},
	Response: &apimodel.OpenReportResponse{},
}, {
	Name:   "SubmitMeasurement",
	Method: "POST",
	URLPath: URLPath{
		InSwagger:  "/report/{report_id}",
		IsTemplate: true,
		Value:      "/report/{{ .ReportID }}",
	},
	Request:  &apimodel.SubmitMeasurementRequest{},
	Response: &apimodel.SubmitMeasurementResponse{},
}}
