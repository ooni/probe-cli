package main

import "github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"

// URLPath describes a URLPath.
type URLPath struct {
	// IsTemplate indicates whether Value contains a template. A future
	// version of this implementation will automatically deduce that.
	IsTemplate bool

	// Value is the value of the URL path.
	Value string

	// InSwagger indicates the corresponding name to be used in
	// the Swagger specification.
	InSwagger string
}

// Descriptor is an API descriptor. It tells the generator
// what code it should emit for a given API.
type Descriptor struct {
	// Name is the name of the API.
	Name string

	// CachePolicy indicates the caching policy to use.
	CachePolicy int

	// RequiresLogin indicates whether the API requires login.
	RequiresLogin bool

	// Method is the method to use ("GET" or "POST").
	Method string

	// URLPath is the URL path.
	URLPath URLPath

	// Request is an instance of the request type.
	Request interface{}

	// Response is an instance of the response type.
	Response interface{}
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

// Descriptors describes all the APIs.
//
// Note that it matters whether the requests and responses
// are pointers. Generally speaking, if the message is a
// struct, use a pointer. If it's a map, don't.
var Descriptors = []Descriptor{{
	Name:     "CheckReportID",
	Method:   "GET",
	URLPath:  URLPath{Value: "/api/_/check_report_id"},
	Request:  &apimodel.CheckReportIDRequest{},
	Response: &apimodel.CheckReportIDResponse{},
}, {
	Name:     "CheckIn",
	Method:   "POST",
	URLPath:  URLPath{Value: "/api/v1/check-in"},
	Request:  &apimodel.CheckInRequest{},
	Response: &apimodel.CheckInResponse{},
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
	Name:     "TestHelpers",
	Method:   "GET",
	URLPath:  URLPath{Value: "/api/v1/test-helpers"},
	Request:  &apimodel.TestHelpersRequest{},
	Response: apimodel.TestHelpersResponse{},
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
}, {
	Name:     "URLs",
	Method:   "GET",
	URLPath:  URLPath{Value: "/api/v1/test-list/urls"},
	Request:  &apimodel.URLsRequest{},
	Response: &apimodel.URLsResponse{},
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
