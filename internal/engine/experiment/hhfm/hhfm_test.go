package hhfm_test

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/hhfm"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	if measurer.ExperimentName() != "http_header_field_manipulation" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.0" {
		t.Fatal("unexpected version")
	}
}

func TestSuccess(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": {{
				Address: "http://37.218.241.94:80",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if tk.Failure != nil {
		t.Fatal("invalid Failure", *tk.Failure)
	}
	if len(tk.Requests) != 1 {
		t.Fatal("invalid Requests")
	}
	request := tk.Requests[0]
	if request.Failure != nil {
		t.Fatal("invalid Requests[0].Failure")
	}
	if request.Request.Body.Value != "" {
		t.Fatal("invalid Requests[0].Request.Body.Value")
	}
	if request.Request.BodyIsTruncated != false {
		t.Fatal("invalid Requests[0].Request.BodyIsTruncated")
	}
	if len(request.Request.HeadersList) != 6 {
		t.Fatal("invalid Requests[0].Request.HeadersList length")
	}
	if len(request.Request.Headers) != 6 {
		t.Fatal("invalid Requests[0].Request.Headers length")
	}
	if strings.ToUpper(request.Request.Method) != "GET" {
		t.Fatal("invalid Requests[0].Request.Method")
	}
	if request.Request.Tor.ExitIP != nil {
		t.Fatal("invalid Requests[0].Request.Tor.ExitIP")
	}
	if request.Request.Tor.ExitName != nil {
		t.Fatal("invalid Requests[0].Request.Tor.ExitName")
	}
	if request.Request.Tor.IsTor != false {
		t.Fatal("invalid Requests[0].Request.Tor.IsTor")
	}
	ths, ok := sess.GetTestHelpersByName("http-return-json-headers")
	if !ok || len(ths) < 1 || ths[0].Type != "legacy" {
		t.Fatal("cannot get the test helper")
	}
	if request.Request.URL != ths[0].Address {
		t.Fatal("invalid Requests[0].Request.URL")
	}
	if len(request.Response.Body.Value) < 1 {
		t.Fatal("invalid Requests[0].Response.Body.Value length")
	}
	if request.Response.BodyIsTruncated != false {
		t.Fatal("invalid Requests[0].Response.BodyIsTruncated")
	}
	if request.Response.Code != 200 {
		t.Fatal("invalid Requests[0].Code")
	}
	if len(request.Response.HeadersList) != 0 {
		t.Fatal("invalid Requests[0].HeadersList length")
	}
	if len(request.Response.Headers) != 0 {
		t.Fatal("invalid Requests[0].Headers length")
	}
	if request.T != 0 {
		t.Fatal("invalid Requests[0].T")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != false {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestCancelledContext(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sess := &mockable.Session{
		MockableLogger: log.Log,
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": {{
				Address: "http://37.218.241.94:80",
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if *tk.Failure != errorsx.FailureInterrupted {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("invalid Requests")
	}
	request := tk.Requests[0]
	if *request.Failure != errorsx.FailureInterrupted {
		t.Fatal("invalid Requests[0].Failure")
	}
	if request.Request.Body.Value != "" {
		t.Fatal("invalid Requests[0].Request.Body.Value")
	}
	if request.Request.BodyIsTruncated != false {
		t.Fatal("invalid Requests[0].Request.BodyIsTruncated")
	}
	if len(request.Request.HeadersList) != 6 {
		t.Fatal("invalid Requests[0].Request.HeadersList length")
	}
	if len(request.Request.Headers) != 6 {
		t.Fatal("invalid Requests[0].Request.Headers length")
	}
	if strings.ToUpper(request.Request.Method) != "GET" {
		t.Fatal("invalid Requests[0].Request.Method")
	}
	if request.Request.Tor.ExitIP != nil {
		t.Fatal("invalid Requests[0].Request.Tor.ExitIP")
	}
	if request.Request.Tor.ExitName != nil {
		t.Fatal("invalid Requests[0].Request.Tor.ExitName")
	}
	if request.Request.Tor.IsTor != false {
		t.Fatal("invalid Requests[0].Request.Tor.IsTor")
	}
	ths, ok := sess.GetTestHelpersByName("http-return-json-headers")
	if !ok || len(ths) < 1 || ths[0].Type != "legacy" {
		t.Fatal("cannot get the test helper")
	}
	if request.Request.URL != ths[0].Address {
		t.Fatal("invalid Requests[0].Request.URL")
	}
	if len(request.Response.Body.Value) != 0 {
		t.Fatal("invalid Requests[0].Response.Body.Value length")
	}
	if request.Response.BodyIsTruncated != false {
		t.Fatal("invalid Requests[0].Response.BodyIsTruncated")
	}
	if request.Response.Code != 0 {
		t.Fatal("invalid Requests[0].Code")
	}
	if len(request.Response.HeadersList) != 0 {
		t.Fatal("invalid Requests[0].HeadersList length")
	}
	if len(request.Response.Headers) != 0 {
		t.Fatal("invalid Requests[0].Headers length")
	}
	if request.T != 0 {
		t.Fatal("invalid Requests[0].T")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != true {
		t.Fatal("invalid Tampering.Total")
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(hhfm.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestNoHelpers(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, hhfm.ErrNoAvailableTestHelpers) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if tk.Failure != nil {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("invalid Requests")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != false {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestNoActualHelpersInList(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": nil,
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, hhfm.ErrNoAvailableTestHelpers) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if tk.Failure != nil {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("invalid Requests")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != false {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestWrongTestHelperType(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": {{
				Address: "http://127.0.0.1",
				Type:    "antani",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if !errors.Is(err, hhfm.ErrInvalidHelperType) {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if tk.Failure != nil {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("invalid Requests")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != false {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestNewRequestFailure(t *testing.T) {
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": {{
				Address: "http://127.0.0.1\t\t\t", // invalid
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if tk.Failure != nil {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("invalid Requests")
	}
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != false {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestInvalidJSONBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello, client") // not valid JSON
	}))
	defer server.Close()
	measurer := hhfm.NewExperimentMeasurer(hhfm.Config{})
	ctx := context.Background()
	sess := &mockable.Session{
		MockableTestHelpers: map[string][]model.Service{
			"http-return-json-headers": {{
				Address: server.URL,
				Type:    "legacy",
			}},
		},
	}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*hhfm.TestKeys)
	if tk.Agent != "agent" {
		t.Fatal("invalid Agent")
	}
	if *tk.Failure != errorsx.FailureJSONParseError {
		t.Fatal("invalid Failure")
	}
	if len(tk.Requests) != 1 {
		t.Fatal("invalid Requests")
	}
	// we already check the content of Requests in other tests
	if tk.SOCKSProxy != nil {
		t.Fatal("invalid SOCKSProxy")
	}
	if tk.Tampering.HeaderFieldName != false {
		t.Fatal("invalid Tampering.HeaderFieldName")
	}
	if tk.Tampering.HeaderFieldNumber != false {
		t.Fatal("invalid Tampering.HeaderFieldNumber")
	}
	if tk.Tampering.HeaderFieldValue != false {
		t.Fatal("invalid Tampering.HeaderFieldValue")
	}
	if tk.Tampering.HeaderNameCapitalization != false {
		t.Fatal("invalid Tampering.HeaderNameCapitalization")
	}
	if len(tk.Tampering.HeaderNameDiff) != 0 {
		t.Fatal("invalid Tampering.HeaderNameDiff")
	}
	if tk.Tampering.RequestLineCapitalization != false {
		t.Fatal("invalid Tampering.RequestLineCapitalization")
	}
	if tk.Tampering.Total != true {
		t.Fatal("invalid Tampering.Total")
	}
}

func TestTransactStatusCodeFailure(t *testing.T) {
	txp := FakeTransport{Resp: &http.Response{
		Body:       io.NopCloser(strings.NewReader("")),
		StatusCode: 500,
	}}
	resp, body, err := hhfm.Transact(txp, &http.Request{},
		model.NewPrinterCallbacks(log.Log))
	if !errors.Is(err, urlgetter.ErrHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("resp is not nil")
	}
	if body != nil {
		t.Fatal("body is not nil")
	}
}

func TestTransactCannotReadBody(t *testing.T) {
	expected := errors.New("mocked error")
	txp := FakeTransport{Resp: &http.Response{
		Body:       &FakeBody{Err: expected},
		StatusCode: 200,
	}}
	resp, body, err := hhfm.Transact(txp, &http.Request{},
		model.NewPrinterCallbacks(log.Log))
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("resp is not nil")
	}
	if body != nil {
		t.Fatal("body is not nil")
	}
}

func TestTestKeys_FillTampering(t *testing.T) {
	type fields struct {
		Agent      string
		Failure    *string
		Requests   []archival.RequestEntry
		SOCKSProxy *string
		Tampering  hhfm.Tampering
	}
	type args struct {
		req         *http.Request
		jsonHeaders hhfm.JSONHeaders
		headers     map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{{
		name: "Request line capitalisation",
		fields: fields{
			Tampering: hhfm.Tampering{
				RequestLineCapitalization: true,
			},
		},
		args: args{
			req: &http.Request{
				Method: "GeT",
			},
			jsonHeaders: hhfm.JSONHeaders{
				RequestLine: "GET / HTTP/1.1",
			},
		},
	}, {
		name: "Header field number",
		fields: fields{
			Tampering: hhfm.Tampering{
				HeaderFieldNumber: true,
			},
		},
		args: args{
			req: &http.Request{
				Method: "GeT",
			},
			jsonHeaders: hhfm.JSONHeaders{
				RequestLine: "GeT / HTTP/1.1",
			},
			headers: map[string]string{
				"UsEr-AgENt": "miniooni/0.1.0-dev",
			},
		},
	}, {
		name: "Header name diff",
		fields: fields{
			Tampering: hhfm.Tampering{
				HeaderNameCapitalization: true,
				HeaderNameDiff:           []string{"UsEr-AgENt", "User-Agent"},
			},
		},
		args: args{
			req: &http.Request{
				Method: "GeT",
			},
			jsonHeaders: hhfm.JSONHeaders{
				RequestLine: "GeT / HTTP/1.1",
				HeadersDict: map[string][]string{
					"User-Agent": {"miniooni/0.1.0-dev"},
				},
			},
			headers: map[string]string{
				"UsEr-AgENt": "miniooni/0.1.0-dev",
			},
		},
	}, {
		name: "Header value diff",
		fields: fields{
			Tampering: hhfm.Tampering{
				HeaderFieldValue: true,
			},
		},
		args: args{
			req: &http.Request{
				Method: "GeT",
			},
			jsonHeaders: hhfm.JSONHeaders{
				RequestLine: "GeT / HTTP/1.1",
				HeadersDict: map[string][]string{
					"UsEr-AgENt": {"MINIOONI/0.1.0-dev"},
				},
			},
			headers: map[string]string{
				"UsEr-AgENt": "miniooni/0.1.0-dev",
			},
		},
	}, {
		name: "Number of headers per key diffs",
		fields: fields{
			Tampering: hhfm.Tampering{
				HeaderFieldValue: true,
			},
		},
		args: args{
			req: &http.Request{
				Method: "GeT",
			},
			jsonHeaders: hhfm.JSONHeaders{
				RequestLine: "GeT / HTTP/1.1",
				HeadersDict: map[string][]string{
					"UsEr-AgENt": {"miniooni/0.1.0-dev", "ooniprobe-engine/0.1.0-dev"},
				},
			},
			headers: map[string]string{
				"UsEr-AgENt": "miniooni/0.1.0-dev",
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &hhfm.TestKeys{
				Agent:      tt.fields.Agent,
				Failure:    tt.fields.Failure,
				Requests:   tt.fields.Requests,
				SOCKSProxy: tt.fields.SOCKSProxy,
			}
			tk.FillTampering(tt.args.req, tt.args.jsonHeaders, tt.args.headers)
			if diff := cmp.Diff(tt.fields.Tampering, tk.Tampering); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewRequestEntryList(t *testing.T) {
	type args struct {
		req     *http.Request
		headers map[string]string
	}
	tests := []struct {
		name    string
		args    args
		wantOut []archival.RequestEntry
	}{{
		name: "common case",
		args: args{
			req: &http.Request{
				Method: "GeT",
				URL: &url.URL{
					Scheme: "http",
					Host:   "10.0.0.1",
					Path:   "/",
				},
			},
			headers: map[string]string{
				"ContENt-tYPE": "text/plain",
				"User-aGENT":   "foo/1.0",
			},
		},
		wantOut: []archival.RequestEntry{{
			Request: archival.HTTPRequest{
				HeadersList: []archival.HTTPHeader{{
					Key:   "ContENt-tYPE",
					Value: archival.MaybeBinaryValue{Value: "text/plain"},
				}, {
					Key:   "User-aGENT",
					Value: archival.MaybeBinaryValue{Value: "foo/1.0"},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
					"ContENt-tYPE": {Value: "text/plain"},
					"User-aGENT":   {Value: "foo/1.0"},
				},
				Method: "GeT",
				URL:    "http://10.0.0.1/",
			},
		}},
	}, {
		name: "without headers",
		args: args{
			req: &http.Request{
				Method: "GeT",
				URL: &url.URL{
					Scheme: "http",
					Host:   "10.0.0.1",
					Path:   "/",
				},
			},
		},
		wantOut: []archival.RequestEntry{{
			Request: archival.HTTPRequest{
				Method:      "GeT",
				Headers:     make(map[string]archival.MaybeBinaryValue),
				HeadersList: []archival.HTTPHeader{},
				URL:         "http://10.0.0.1/",
			},
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := hhfm.NewRequestEntryList(tt.args.req, tt.args.headers)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewHTTPResponse(t *testing.T) {
	type args struct {
		resp *http.Response
		data []byte
	}
	tests := []struct {
		name    string
		args    args
		wantOut archival.HTTPResponse
	}{{
		name: "common case",
		args: args{
			resp: &http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type": []string{"text/plain"},
					"User-Agent":   []string{"foo/1.0"},
				},
			},
			data: []byte("deadbeef"),
		},
		wantOut: archival.HTTPResponse{
			Body: archival.MaybeBinaryValue{Value: "deadbeef"},
			Code: 200,
			HeadersList: []archival.HTTPHeader{{
				Key:   "Content-Type",
				Value: archival.MaybeBinaryValue{Value: "text/plain"},
			}, {
				Key:   "User-Agent",
				Value: archival.MaybeBinaryValue{Value: "foo/1.0"},
			}},
			Headers: map[string]archival.MaybeBinaryValue{
				"Content-Type": {Value: "text/plain"},
				"User-Agent":   {Value: "foo/1.0"},
			},
		},
	}, {
		name: "with no HTTP header and body",
		args: args{
			resp: &http.Response{StatusCode: 200},
		},
		wantOut: archival.HTTPResponse{
			Body:        archival.MaybeBinaryValue{Value: ""},
			Code:        200,
			HeadersList: []archival.HTTPHeader{},
			Headers:     map[string]archival.MaybeBinaryValue{},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := hhfm.NewHTTPResponse(tt.args.resp, tt.args.data)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestDialerDialContext(t *testing.T) {
	expected := errors.New("mocked error")
	d := hhfm.Dialer{Dialer: FakeDialer{Err: expected}}
	conn, err := d.DialContext(context.Background(), "tcp", "127.0.0.1:80")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("conn is not nil")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &hhfm.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	tests := []struct {
		tampering hhfm.Tampering
		isAnomaly bool
	}{{
		tampering: hhfm.Tampering{},
		isAnomaly: false,
	}, {
		tampering: hhfm.Tampering{HeaderFieldName: true},
		isAnomaly: true,
	}, {
		tampering: hhfm.Tampering{HeaderFieldNumber: true},
		isAnomaly: true,
	}, {
		tampering: hhfm.Tampering{HeaderFieldValue: true},
		isAnomaly: true,
	}, {
		tampering: hhfm.Tampering{HeaderNameCapitalization: true},
		isAnomaly: true,
	}, {
		tampering: hhfm.Tampering{RequestLineCapitalization: true},
		isAnomaly: true,
	}, {
		tampering: hhfm.Tampering{Total: true},
		isAnomaly: true,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &hhfm.Measurer{}
			measurement := &model.Measurement{TestKeys: &hhfm.TestKeys{
				Tampering: tt.tampering,
			}}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(hhfm.SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
		})
	}
}
