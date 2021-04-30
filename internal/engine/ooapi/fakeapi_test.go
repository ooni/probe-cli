// Code generated by go generate; DO NOT EDIT.
// 2021-04-26 14:35:20.569073374 +0200 CEST m=+0.000071055

package ooapi

//go:generate go run ./internal/generator -file fakeapi_test.go

import (
	"context"
	"sync/atomic"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

type FakeCheckReportIDAPI struct {
	Err       error
	Response  *apimodel.CheckReportIDResponse
	CountCall int32
}

func (fapi *FakeCheckReportIDAPI) Call(ctx context.Context, req *apimodel.CheckReportIDRequest) (*apimodel.CheckReportIDResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForCheckReportIDAPI = &FakeCheckReportIDAPI{}
)

type FakeCheckInAPI struct {
	Err       error
	Response  *apimodel.CheckInResponse
	CountCall int32
}

func (fapi *FakeCheckInAPI) Call(ctx context.Context, req *apimodel.CheckInRequest) (*apimodel.CheckInResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForCheckInAPI = &FakeCheckInAPI{}
)

type FakeLoginAPI struct {
	Err       error
	Response  *apimodel.LoginResponse
	CountCall int32
}

func (fapi *FakeLoginAPI) Call(ctx context.Context, req *apimodel.LoginRequest) (*apimodel.LoginResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForLoginAPI = &FakeLoginAPI{}
)

type FakeMeasurementMetaAPI struct {
	Err       error
	Response  *apimodel.MeasurementMetaResponse
	CountCall int32
}

func (fapi *FakeMeasurementMetaAPI) Call(ctx context.Context, req *apimodel.MeasurementMetaRequest) (*apimodel.MeasurementMetaResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForMeasurementMetaAPI = &FakeMeasurementMetaAPI{}
)

type FakeRegisterAPI struct {
	Err       error
	Response  *apimodel.RegisterResponse
	CountCall int32
}

func (fapi *FakeRegisterAPI) Call(ctx context.Context, req *apimodel.RegisterRequest) (*apimodel.RegisterResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForRegisterAPI = &FakeRegisterAPI{}
)

type FakeTestHelpersAPI struct {
	Err       error
	Response  apimodel.TestHelpersResponse
	CountCall int32
}

func (fapi *FakeTestHelpersAPI) Call(ctx context.Context, req *apimodel.TestHelpersRequest) (apimodel.TestHelpersResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForTestHelpersAPI = &FakeTestHelpersAPI{}
)

type FakePsiphonConfigAPI struct {
	WithResult callerForPsiphonConfigAPI
	Err        error
	Response   apimodel.PsiphonConfigResponse
	CountCall  int32
}

func (fapi *FakePsiphonConfigAPI) Call(ctx context.Context, req *apimodel.PsiphonConfigRequest) (apimodel.PsiphonConfigResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

func (fapi *FakePsiphonConfigAPI) WithToken(token string) callerForPsiphonConfigAPI {
	return fapi.WithResult
}

var (
	_ callerForPsiphonConfigAPI = &FakePsiphonConfigAPI{}
	_ clonerForPsiphonConfigAPI = &FakePsiphonConfigAPI{}
)

type FakeTorTargetsAPI struct {
	WithResult callerForTorTargetsAPI
	Err        error
	Response   apimodel.TorTargetsResponse
	CountCall  int32
}

func (fapi *FakeTorTargetsAPI) Call(ctx context.Context, req *apimodel.TorTargetsRequest) (apimodel.TorTargetsResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

func (fapi *FakeTorTargetsAPI) WithToken(token string) callerForTorTargetsAPI {
	return fapi.WithResult
}

var (
	_ callerForTorTargetsAPI = &FakeTorTargetsAPI{}
	_ clonerForTorTargetsAPI = &FakeTorTargetsAPI{}
)

type FakeURLsAPI struct {
	Err       error
	Response  *apimodel.URLsResponse
	CountCall int32
}

func (fapi *FakeURLsAPI) Call(ctx context.Context, req *apimodel.URLsRequest) (*apimodel.URLsResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForURLsAPI = &FakeURLsAPI{}
)

type FakeOpenReportAPI struct {
	Err       error
	Response  *apimodel.OpenReportResponse
	CountCall int32
}

func (fapi *FakeOpenReportAPI) Call(ctx context.Context, req *apimodel.OpenReportRequest) (*apimodel.OpenReportResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForOpenReportAPI = &FakeOpenReportAPI{}
)

type FakeSubmitMeasurementAPI struct {
	Err       error
	Response  *apimodel.SubmitMeasurementResponse
	CountCall int32
}

func (fapi *FakeSubmitMeasurementAPI) Call(ctx context.Context, req *apimodel.SubmitMeasurementRequest) (*apimodel.SubmitMeasurementResponse, error) {
	atomic.AddInt32(&fapi.CountCall, 1)
	return fapi.Response, fapi.Err
}

var (
	_ callerForSubmitMeasurementAPI = &FakeSubmitMeasurementAPI{}
)
