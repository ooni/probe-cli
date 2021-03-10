// Code generated by go generate; DO NOT EDIT.
// 2021-03-10 13:04:07.20568416 +0100 CET m=+0.000149001

package ooapi

//go:generate go run ./internal/generator -file clientcall_test.go

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/ooapi/apimodel"
)

type handleClientCallCheckReportID struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.CheckReportIDResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallCheckReportID) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.CheckReportIDResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestCheckReportIDClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallCheckReportID{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.CheckReportIDRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.CheckReportID(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simpleCheckReportIDAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallCheckIn struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.CheckInResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallCheckIn) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.CheckInResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestCheckInClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallCheckIn{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.CheckInRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.CheckIn(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "POST" {
		t.Fatal("invalid method")
	}
	// check the body
	if handler.contentType != "application/json" {
		t.Fatal("invalid content-type header")
	}
	got := &apimodel.CheckInRequest{}
	if err := json.Unmarshal(handler.body, &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(req, got); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallMeasurementMeta struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.MeasurementMetaResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallMeasurementMeta) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.MeasurementMetaResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestMeasurementMetaClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallMeasurementMeta{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.MeasurementMetaRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.MeasurementMeta(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simpleMeasurementMetaAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallTestHelpers struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        apimodel.TestHelpersResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallTestHelpers) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out apimodel.TestHelpersResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestTestHelpersClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallTestHelpers{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.TestHelpersRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.TestHelpers(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simpleTestHelpersAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallPsiphonConfig struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        apimodel.PsiphonConfigResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallPsiphonConfig) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	if r.URL.Path == "/api/v1/register" {
		var out apimodel.RegisterResponse
		ff.fill(&out)
		data, err := json.Marshal(out)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(data)
		return
	}
	if r.URL.Path == "/api/v1/login" {
		var out apimodel.LoginResponse
		ff.fill(&out)
		data, err := json.Marshal(out)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(data)
		return
	}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out apimodel.PsiphonConfigResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestPsiphonConfigClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallPsiphonConfig{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.PsiphonConfigRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.PsiphonConfig(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simplePsiphonConfigAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallTorTargets struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        apimodel.TorTargetsResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallTorTargets) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	if r.URL.Path == "/api/v1/register" {
		var out apimodel.RegisterResponse
		ff.fill(&out)
		data, err := json.Marshal(out)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(data)
		return
	}
	if r.URL.Path == "/api/v1/login" {
		var out apimodel.LoginResponse
		ff.fill(&out)
		data, err := json.Marshal(out)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		w.Write(data)
		return
	}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out apimodel.TorTargetsResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestTorTargetsClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallTorTargets{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.TorTargetsRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.TorTargets(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simpleTorTargetsAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallURLs struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.URLsResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallURLs) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.URLsResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestURLsClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallURLs{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.URLsRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.URLs(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "GET" {
		t.Fatal("invalid method")
	}
	// check the query
	api := &simpleURLsAPI{BaseURL: srvr.URL}
	httpReq, err := api.newRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(handler.url.Path, httpReq.URL.Path); diff != "" {
		t.Fatal(diff)
	}
	if diff := cmp.Diff(handler.url.RawQuery, httpReq.URL.RawQuery); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallOpenReport struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.OpenReportResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallOpenReport) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.OpenReportResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestOpenReportClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallOpenReport{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.OpenReportRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.OpenReport(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "POST" {
		t.Fatal("invalid method")
	}
	// check the body
	if handler.contentType != "application/json" {
		t.Fatal("invalid content-type header")
	}
	got := &apimodel.OpenReportRequest{}
	if err := json.Unmarshal(handler.body, &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(req, got); diff != "" {
		t.Fatal(diff)
	}
}

type handleClientCallSubmitMeasurement struct {
	accept      string
	body        []byte
	contentType string
	count       int32
	method      string
	mu          sync.Mutex
	resp        *apimodel.SubmitMeasurementResponse
	url         *url.URL
	userAgent   string
}

func (h *handleClientCallSubmitMeasurement) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ff := fakeFill{}
	defer h.mu.Unlock()
	h.mu.Lock()
	if h.count > 0 {
		w.WriteHeader(400)
		return
	}
	h.count++
	if r.Body != nil {
		data, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(400)
			return
		}
		h.body = data
	}
	h.method = r.Method
	h.url = r.URL
	h.accept = r.Header.Get("Accept")
	h.contentType = r.Header.Get("Content-Type")
	h.userAgent = r.Header.Get("User-Agent")
	var out *apimodel.SubmitMeasurementResponse
	ff.fill(&out)
	h.resp = out
	data, err := json.Marshal(out)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	w.Write(data)
}

func TestSubmitMeasurementClientCallRoundTrip(t *testing.T) {
	// setup
	handler := &handleClientCallSubmitMeasurement{}
	srvr := httptest.NewServer(handler)
	defer srvr.Close()
	req := &apimodel.SubmitMeasurementRequest{}
	ff := &fakeFill{}
	ff.fill(&req)
	clnt := &Client{KVStore: &memkvstore{}, BaseURL: srvr.URL}
	ff.fill(&clnt.UserAgent)
	// issue request
	ctx := context.Background()
	resp, err := clnt.SubmitMeasurement(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response here")
	}
	// compare our response and server's one
	if diff := cmp.Diff(handler.resp, resp); diff != "" {
		t.Fatal(diff)
	}
	// check whether headers are OK
	if handler.accept != "application/json" {
		t.Fatal("invalid accept header")
	}
	if handler.userAgent != clnt.UserAgent {
		t.Fatal("invalid user-agent header")
	}
	// check whether the method is OK
	if handler.method != "POST" {
		t.Fatal("invalid method")
	}
	// check the body
	if handler.contentType != "application/json" {
		t.Fatal("invalid content-type header")
	}
	got := &apimodel.SubmitMeasurementRequest{}
	if err := json.Unmarshal(handler.body, &got); err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(req, got); diff != "" {
		t.Fatal(diff)
	}
}
