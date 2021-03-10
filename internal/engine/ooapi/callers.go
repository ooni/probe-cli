// Code generated by go generate; DO NOT EDIT.
// 2021-03-10 12:20:36.709496294 +0100 CET m=+0.000184271

package ooapi

//go:generate go run ./internal/generator -file callers.go

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

// callerForCheckReportIDAPI represents any type exposing a method
// like simpleCheckReportIDAPI.Call.
type callerForCheckReportIDAPI interface {
	Call(ctx context.Context, req *apimodel.CheckReportIDRequest) (*apimodel.CheckReportIDResponse, error)
}

// callerForCheckInAPI represents any type exposing a method
// like simpleCheckInAPI.Call.
type callerForCheckInAPI interface {
	Call(ctx context.Context, req *apimodel.CheckInRequest) (*apimodel.CheckInResponse, error)
}

// callerForLoginAPI represents any type exposing a method
// like simpleLoginAPI.Call.
type callerForLoginAPI interface {
	Call(ctx context.Context, req *apimodel.LoginRequest) (*apimodel.LoginResponse, error)
}

// callerForMeasurementMetaAPI represents any type exposing a method
// like simpleMeasurementMetaAPI.Call.
type callerForMeasurementMetaAPI interface {
	Call(ctx context.Context, req *apimodel.MeasurementMetaRequest) (*apimodel.MeasurementMetaResponse, error)
}

// callerForRegisterAPI represents any type exposing a method
// like simpleRegisterAPI.Call.
type callerForRegisterAPI interface {
	Call(ctx context.Context, req *apimodel.RegisterRequest) (*apimodel.RegisterResponse, error)
}

// callerForTestHelpersAPI represents any type exposing a method
// like simpleTestHelpersAPI.Call.
type callerForTestHelpersAPI interface {
	Call(ctx context.Context, req *apimodel.TestHelpersRequest) (apimodel.TestHelpersResponse, error)
}

// callerForPsiphonConfigAPI represents any type exposing a method
// like simplePsiphonConfigAPI.Call.
type callerForPsiphonConfigAPI interface {
	Call(ctx context.Context, req *apimodel.PsiphonConfigRequest) (apimodel.PsiphonConfigResponse, error)
}

// callerForTorTargetsAPI represents any type exposing a method
// like simpleTorTargetsAPI.Call.
type callerForTorTargetsAPI interface {
	Call(ctx context.Context, req *apimodel.TorTargetsRequest) (apimodel.TorTargetsResponse, error)
}

// callerForURLsAPI represents any type exposing a method
// like simpleURLsAPI.Call.
type callerForURLsAPI interface {
	Call(ctx context.Context, req *apimodel.URLsRequest) (*apimodel.URLsResponse, error)
}

// callerForOpenReportAPI represents any type exposing a method
// like simpleOpenReportAPI.Call.
type callerForOpenReportAPI interface {
	Call(ctx context.Context, req *apimodel.OpenReportRequest) (*apimodel.OpenReportResponse, error)
}

// callerForSubmitMeasurementAPI represents any type exposing a method
// like simpleSubmitMeasurementAPI.Call.
type callerForSubmitMeasurementAPI interface {
	Call(ctx context.Context, req *apimodel.SubmitMeasurementRequest) (*apimodel.SubmitMeasurementResponse, error)
}
