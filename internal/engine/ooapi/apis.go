// Code generated by go generate; DO NOT EDIT.
// 2021-02-23 11:20:33.942538 +0100 CET m=+0.000207444

package ooapi

//go:generate go run ./internal/generator

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

// CheckReportIDAPI is the CheckReportID API.
type CheckReportIDAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *CheckReportIDAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *CheckReportIDAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *CheckReportIDAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *CheckReportIDAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the CheckReportID API.
func (api *CheckReportIDAPI) Call(ctx context.Context, req *apimodel.CheckReportIDRequest) (*apimodel.CheckReportIDResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// CheckInAPI is the CheckIn API.
type CheckInAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *CheckInAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *CheckInAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *CheckInAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *CheckInAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the CheckIn API.
func (api *CheckInAPI) Call(ctx context.Context, req *apimodel.CheckInRequest) (*apimodel.CheckInResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// LoginAPI is the Login API.
type LoginAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *LoginAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *LoginAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *LoginAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *LoginAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the Login API.
func (api *LoginAPI) Call(ctx context.Context, req *apimodel.LoginRequest) (*apimodel.LoginResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// MeasurementMetaAPI is the MeasurementMeta API.
type MeasurementMetaAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *MeasurementMetaAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *MeasurementMetaAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *MeasurementMetaAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *MeasurementMetaAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the MeasurementMeta API.
func (api *MeasurementMetaAPI) Call(ctx context.Context, req *apimodel.MeasurementMetaRequest) (*apimodel.MeasurementMetaResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// RegisterAPI is the Register API.
type RegisterAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *RegisterAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *RegisterAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *RegisterAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *RegisterAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the Register API.
func (api *RegisterAPI) Call(ctx context.Context, req *apimodel.RegisterRequest) (*apimodel.RegisterResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// TestHelpersAPI is the TestHelpers API.
type TestHelpersAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *TestHelpersAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *TestHelpersAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *TestHelpersAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *TestHelpersAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the TestHelpers API.
func (api *TestHelpersAPI) Call(ctx context.Context, req *apimodel.TestHelpersRequest) (apimodel.TestHelpersResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// PsiphonConfigAPI is the PsiphonConfig API.
type PsiphonConfigAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	Token        string       // mandatory
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *PsiphonConfigAPI) WithToken(token string) PsiphonConfigCaller {
	out := &PsiphonConfigAPI{}
	out.BaseURL = api.BaseURL
	out.HTTPClient = api.HTTPClient
	out.JSONCodec = api.JSONCodec
	out.RequestMaker = api.RequestMaker
	out.UserAgent = api.UserAgent
	out.Token = token
	return out
}

func (api *PsiphonConfigAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *PsiphonConfigAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *PsiphonConfigAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *PsiphonConfigAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the PsiphonConfig API.
func (api *PsiphonConfigAPI) Call(ctx context.Context, req *apimodel.PsiphonConfigRequest) (apimodel.PsiphonConfigResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	if api.Token == "" {
		return nil, ErrMissingToken
	}
	httpReq.Header.Add("Authorization", newAuthorizationHeader(api.Token))
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// TorTargetsAPI is the TorTargets API.
type TorTargetsAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	Token        string       // mandatory
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *TorTargetsAPI) WithToken(token string) TorTargetsCaller {
	out := &TorTargetsAPI{}
	out.BaseURL = api.BaseURL
	out.HTTPClient = api.HTTPClient
	out.JSONCodec = api.JSONCodec
	out.RequestMaker = api.RequestMaker
	out.UserAgent = api.UserAgent
	out.Token = token
	return out
}

func (api *TorTargetsAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *TorTargetsAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *TorTargetsAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *TorTargetsAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the TorTargets API.
func (api *TorTargetsAPI) Call(ctx context.Context, req *apimodel.TorTargetsRequest) (apimodel.TorTargetsResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	if api.Token == "" {
		return nil, ErrMissingToken
	}
	httpReq.Header.Add("Authorization", newAuthorizationHeader(api.Token))
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// URLsAPI is the URLs API.
type URLsAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *URLsAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *URLsAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *URLsAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *URLsAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the URLs API.
func (api *URLsAPI) Call(ctx context.Context, req *apimodel.URLsRequest) (*apimodel.URLsResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// OpenReportAPI is the OpenReport API.
type OpenReportAPI struct {
	BaseURL      string       // optional
	HTTPClient   HTTPClient   // optional
	JSONCodec    JSONCodec    // optional
	RequestMaker RequestMaker // optional
	UserAgent    string       // optional
}

func (api *OpenReportAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *OpenReportAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *OpenReportAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *OpenReportAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the OpenReport API.
func (api *OpenReportAPI) Call(ctx context.Context, req *apimodel.OpenReportRequest) (*apimodel.OpenReportResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}

// SubmitMeasurementAPI is the SubmitMeasurement API.
type SubmitMeasurementAPI struct {
	BaseURL          string           // optional
	HTTPClient       HTTPClient       // optional
	JSONCodec        JSONCodec        // optional
	RequestMaker     RequestMaker     // optional
	TemplateExecutor TemplateExecutor // optional
	UserAgent        string           // optional
}

func (api *SubmitMeasurementAPI) baseURL() string {
	if api.BaseURL != "" {
		return api.BaseURL
	}
	return "https://ps1.ooni.io"
}

func (api *SubmitMeasurementAPI) requestMaker() RequestMaker {
	if api.RequestMaker != nil {
		return api.RequestMaker
	}
	return &defaultRequestMaker{}
}

func (api *SubmitMeasurementAPI) jsonCodec() JSONCodec {
	if api.JSONCodec != nil {
		return api.JSONCodec
	}
	return &defaultJSONCodec{}
}

func (api *SubmitMeasurementAPI) templateExecutor() TemplateExecutor {
	if api.TemplateExecutor != nil {
		return api.TemplateExecutor
	}
	return &defaultTemplateExecutor{}
}

func (api *SubmitMeasurementAPI) httpClient() HTTPClient {
	if api.HTTPClient != nil {
		return api.HTTPClient
	}
	return http.DefaultClient
}

// Call calls the SubmitMeasurement API.
func (api *SubmitMeasurementAPI) Call(ctx context.Context, req *apimodel.SubmitMeasurementRequest) (*apimodel.SubmitMeasurementResponse, error) {
	httpReq, err := api.newRequest(ctx, req)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Add("Accept", "application/json")
	httpReq.Header.Add("User-Agent", api.UserAgent)
	return api.newResponse(api.httpClient().Do(httpReq))
}
