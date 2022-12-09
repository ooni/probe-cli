package httpapi

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"syscall"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func Test_joinURLPath(t *testing.T) {
	tests := []struct {
		name         string
		urlPath      string
		resourcePath string
		want         string
	}{{
		name:         "whole path inside urlPath and empty resourcePath",
		urlPath:      "/robots.txt",
		resourcePath: "",
		want:         "/robots.txt",
	}, {
		name:         "empty urlPath and slash-prefixed resourcePath",
		urlPath:      "",
		resourcePath: "/foo",
		want:         "/foo",
	}, {
		name:         "slash urlPath and slash-prefixed resourcePath",
		urlPath:      "/",
		resourcePath: "/foo",
		want:         "/foo",
	}, {
		name:         "empty urlPath and empty resourcePath",
		urlPath:      "",
		resourcePath: "",
		want:         "/",
	}, {
		name:         "non-slash-terminated urlPath and slash-prefixed resourcePath",
		urlPath:      "/foo",
		resourcePath: "/bar",
		want:         "/foo/bar",
	}, {
		name:         "slash-terminated urlPath and slash-prefixed resourcePath",
		urlPath:      "/foo/",
		resourcePath: "/bar",
		want:         "/foo/bar",
	}, {
		name:         "slash-terminated urlPath and non-slash-prefixed resourcePath",
		urlPath:      "/foo",
		resourcePath: "bar",
		want:         "/foo/bar",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinURLPath(tt.urlPath, tt.resourcePath)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func Test_newRequest(t *testing.T) {
	type args struct {
		ctx      context.Context
		endpoint *Endpoint
		desc     *Descriptor
	}
	tests := []struct {
		name    string
		args    args
		wantFn  func(*testing.T, *http.Request)
		wantErr error
	}{{
		name: "url.Parse fails",
		args: args{
			ctx: nil,
			endpoint: &Endpoint{
				BaseURL:    "\t\t\t", // does not parse!
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn:  nil,
		wantErr: errors.New(`parse "\t\t\t": net/url: invalid control character in URL`),
	}, {
		name: "http.NewRequestWithContext fails",
		args: args{
			ctx: nil, // causes http.NewRequestWithContext to fail
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn:  nil,
		wantErr: errors.New("net/http: nil Context"),
	}, {
		name: "successful case with GET method, no body, and no extra headers",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			if req.Body != nil {
				t.Fatal("invalid body", req.Body)
			}
		},
		wantErr: nil,
	}, {
		name: "successful case with POST method and body",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "",
				Logger:     model.DiscardLogger,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodPost,
				RequestBody:   []byte("deadbeef"),
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodPost {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			data, err := netxlite.ReadAllContext(context.Background(), req.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff([]byte("deadbeef"), data); diff != "" {
				t.Fatal(diff)
			}
		},
		wantErr: nil,
	}, {
		name: "with GET method and custom headers",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.com/",
				HTTPClient: nil,
				Host:       "antani.org",
				Logger:     nil,
				UserAgent:  "httpclient/1.0.1",
			},
			desc: &Descriptor{
				Accept:        "application/json",
				Authorization: "deafbeef",
				ContentType:   "text/plain",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodPut,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodPut {
				t.Fatal("invalid method")
			}
			if req.Host != "antani.org" {
				t.Fatal("invalid request host")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
			if req.Header.Get("Authorization") != "deafbeef" {
				t.Fatal("invalid authorization")
			}
			if req.Header.Get("Content-Type") != "text/plain" {
				t.Fatal("invalid content-type")
			}
			if req.Header.Get("Accept") != "application/json" {
				t.Fatal("invalid accept")
			}
			if req.Header.Get("User-Agent") != "httpclient/1.0.1" {
				t.Fatal("invalid user-agent")
			}
		},
		wantErr: nil,
	}, {
		name: "we join the urlPath with the resourcePath",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://www.example.com/api/v1",
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "/test-list/urls",
				URLQuery:      nil,
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://www.example.com/api/v1/test-list/urls" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "we discard any query element inside the Endpoint.BaseURL",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://example.org/api/v1/?probe_cc=IT",
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.org/api/v1/" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "we include query elements from Descriptor.URLQuery",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL:    "https://www.example.com/api/v1/",
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        http.MethodGet,
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "test-list/urls",
				URLQuery: map[string][]string{
					"probe_cc": {"IT"},
				},
			},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://www.example.com/api/v1/test-list/urls?probe_cc=IT" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}, {
		name: "with as many implicitly-initialized fields as possible",
		args: args{
			ctx: context.Background(),
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
			},
			desc: &Descriptor{},
		},
		wantFn: func(t *testing.T, req *http.Request) {
			if req == nil {
				t.Fatal("expected non-nil request")
			}
			if req.Method != http.MethodGet {
				t.Fatal("invalid method")
			}
			if req.URL.String() != "https://example.com/" {
				t.Fatal("invalid URL")
			}
		},
		wantErr: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newRequest(tt.args.ctx, tt.args.endpoint, tt.args.desc)
			switch {
			case err == nil && tt.wantErr == nil:
				// nothing
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
			if tt.wantFn != nil {
				tt.wantFn(t, got)
				return
			}
			if got != nil {
				t.Fatal("got response with nil tt.wantFn")
			}
		})
	}
}

// gzipBombForCall contains one megabyte of zeroes compressed using gzip
var gzipBombForCall = []byte{
	0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0xff, 0xec, 0xc0, 0x31, 0x01, 0x00, 0x00,
	0x00, 0xc2, 0x20, 0xfb, 0xa7, 0x36, 0xc4, 0x5e,
	0x18, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
	0x40, 0xf4, 0x00, 0x00, 0x00, 0xff, 0xff, 0x1c,
	0xea, 0x38, 0xa7, 0x00, 0x00, 0x10, 0x00,
}

func Test_docall(t *testing.T) {
	type args struct {
		endpoint *Endpoint
		desc     *Descriptor
		request  *http.Request
	}
	tests := []struct {
		name     string
		args     args
		wantResp *http.Response
		wantBody []byte
		wantErr  error
	}{{
		name: "we honour the configured max body size",
		args: args{
			endpoint: &Endpoint{
				BaseURL: "http://127.0.0.2/", // actually unused
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader("AAAAAAAAAAAAAAAAA")),
						}
						return resp, nil
					},
				},
				Host:      "",
				Logger:    model.DiscardLogger,
				UserAgent: "",
			},
			desc: &Descriptor{
				MaxBodySize: 7,
				Method:      http.MethodGet,
				URLPath:     "/",
			},
			request: &http.Request{},
		},
		wantResp: &http.Response{
			// Implementation note: the test will ONLY match
			// the status code and the response headers.
			StatusCode: 200,
		},
		wantBody: nil,
		wantErr:  ErrTruncated,
	}, {
		name: "we have a default max body size",
		args: args{
			endpoint: &Endpoint{
				BaseURL: "http://127.0.0.2/", // actually unused
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(strings.NewReader("AAAAAAAAAAAAAAAAA")),
						}
						return resp, nil
					},
				},
				Host:      "",
				Logger:    model.DiscardLogger,
				UserAgent: "",
			},
			desc: &Descriptor{
				MaxBodySize: 0, // means use the default
				Method:      http.MethodGet,
				URLPath:     "/",
			},
			request: &http.Request{},
		},
		wantResp: &http.Response{
			// Implementation note: the test will ONLY match
			// the status code and the response headers.
			StatusCode: 200,
		},
		wantBody: []byte("AAAAAAAAAAAAAAAAA"),
		wantErr:  nil,
	}, {
		name: "we decompress gzip encoded bodies",
		args: args{
			endpoint: &Endpoint{
				BaseURL: "http://127.0.0.2/", // actually unused
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: 200,
							Body: io.NopCloser(bytes.NewReader([]byte{
								0x1f, 0x8b, 0x08, 0x00, 0x00, 0x00, 0x00, 0x00,
								0x00, 0xff, 0x72, 0x74, 0x74, 0x74, 0xd4, 0x45,
								0x25, 0x00, 0x01, 0x00, 0x00, 0xff, 0xff, 0xc2,
								0x43, 0xb0, 0x08, 0x13, 0x00, 0x00, 0x00,
							})),
							Header: http.Header{
								"Content-Encoding": {"gzip"},
							},
						}
						return resp, nil
					},
				},
				Host:      "",
				Logger:    model.DiscardLogger,
				UserAgent: "",
			},
			desc: &Descriptor{
				MaxBodySize: 0, // means use the default
				Method:      http.MethodGet,
				URLPath:     "/",
			},
			request: &http.Request{},
		},
		wantResp: &http.Response{
			// Implementation note: the test will ONLY match
			// the status code and the response headers.
			StatusCode: 200,
			Header: http.Header{
				"Content-Encoding": {"gzip"},
			},
		},
		wantBody: []byte("AAAA-AAAA-AAAA-AAAA"),
		wantErr:  nil,
	}, {
		name: "we handle issues with the gzip header",
		args: args{
			endpoint: &Endpoint{
				BaseURL: "http://127.0.0.2/", // actually unused
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: 200,
							Body: io.NopCloser(bytes.NewReader([]byte{
								0x00, 0x00, 0x00, 0xff, 0xff, 0xff, 0xff, 0xff, // <- changed this line
								0x00, 0xff, 0x72, 0x74, 0x74, 0x74, 0xd4, 0x45,
								0x25, 0x00, 0x01, 0x00, 0x00, 0xff, 0xff, 0xc2,
								0x43, 0xb0, 0x08, 0x13, 0x00, 0x00, 0x00,
							})),
							Header: http.Header{
								"Content-Encoding": {"gzip"},
							},
						}
						return resp, nil
					},
				},
				Host:      "",
				Logger:    model.DiscardLogger,
				UserAgent: "",
			},
			desc: &Descriptor{
				MaxBodySize: 0, // means use the default
				Method:      http.MethodGet,
				URLPath:     "/",
			},
			request: &http.Request{},
		},
		wantResp: &http.Response{
			// Implementation note: the test will ONLY match
			// the status code and the response headers.
			StatusCode: 200,
			Header: http.Header{
				"Content-Encoding": {"gzip"},
			},
		},
		wantBody: nil,
		wantErr:  gzip.ErrHeader,
	}, {
		name: "we protect against a gzip bomb",
		args: args{
			endpoint: &Endpoint{
				BaseURL: "http://127.0.0.2/", // actually unused
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							StatusCode: 200,
							Body:       io.NopCloser(bytes.NewReader(gzipBombForCall)),
							Header: http.Header{
								"Content-Encoding": {"gzip"},
							},
						}
						return resp, nil
					},
				},
				Host:      "",
				Logger:    model.DiscardLogger,
				UserAgent: "",
			},
			desc: &Descriptor{
				MaxBodySize: 2048, // very small value
				Method:      http.MethodGet,
				URLPath:     "/",
			},
			request: &http.Request{},
		},
		wantResp: &http.Response{
			// Implementation note: the test will ONLY match
			// the status code and the response headers.
			StatusCode: 200,
			Header: http.Header{
				"Content-Encoding": {"gzip"},
			},
		},
		wantBody: nil,
		wantErr:  ErrTruncated,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, body, err := docall(tt.args.endpoint, tt.args.desc, tt.args.request)
			if err != tt.wantErr {
				t.Errorf("docall() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(resp.StatusCode, tt.wantResp.StatusCode) {
				t.Errorf("docall() got = %v, want %v", resp.StatusCode, tt.wantResp.StatusCode)
			}
			if !reflect.DeepEqual(resp.Header, tt.wantResp.Header) {
				t.Errorf("docall() got = %v, want %v", resp.Header, tt.wantResp.Header)
			}
			if !reflect.DeepEqual(body, tt.wantBody) {
				t.Errorf("docall() got1 = %v, want %v", body, tt.wantBody)
			}
		})
	}
}

func TestCall(t *testing.T) {
	type args struct {
		ctx      context.Context
		desc     *Descriptor
		endpoint *Endpoint
	}
	tests := []struct {
		name    string
		args    args
		want    []byte
		wantErr error
		errfn   func(t *testing.T, err error)
	}{{
		name: "newRequest fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Accept:        "",
				Authorization: "",
				ContentType:   "",
				LogBody:       false,
				MaxBodySize:   0,
				Method:        "",
				RequestBody:   nil,
				Timeout:       0,
				URLPath:       "",
				URLQuery:      nil,
			},
			endpoint: &Endpoint{
				BaseURL:    "\t\t\t", // causes newRequest to fail
				HTTPClient: nil,
				Host:       "",
				Logger:     nil,
				UserAgent:  "",
			},
		},
		want:    nil,
		wantErr: errors.New(`parse "\t\t\t": net/url: invalid control character in URL`),
		errfn:   nil,
	}, {
		name: "endpoint.HTTPClient.Do fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		want:    nil,
		wantErr: io.EOF,
		errfn: func(t *testing.T, err error) {
			var expect *errMaybeCensorship
			if !errors.As(err, &expect) {
				t.Fatal("unexpected error type")
			}
		},
	}, {
		name: "reading body fails",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									return 0, netxlite.ECONNRESET
								},
							}),
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		want:    nil,
		wantErr: errors.New(netxlite.FailureConnectionReset),
		errfn: func(t *testing.T, err error) {
			var expect *errMaybeCensorship
			if !errors.As(err, &expect) {
				t.Fatal("unexpected error type")
			}
		},
	}, {
		name: "status code indicates failure",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				Method: http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader("deadbeef")),
							StatusCode: 403,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		want:    nil,
		wantErr: errors.New("httpapi: http request failed: 403"),
		errfn: func(t *testing.T, err error) {
			var expect *ErrHTTPRequestFailed
			if !errors.As(err, &expect) {
				t.Fatal("invalid error type")
			}
		},
	}, {
		name: "success with log body flag",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				LogBody: true, // as documented by this test's name
				Method:  http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader("deadbeef")),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		want:    []byte("deadbeef"),
		wantErr: nil,
		errfn:   nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Call(tt.args.ctx, tt.args.desc, tt.args.endpoint)
			switch {
			case err == nil && tt.wantErr == nil:
				// nothing
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestCallWithJSONResponse(t *testing.T) {
	type response struct {
		Name string
		Age  int64
	}
	expectedResponse := response{
		Name: "sbs",
		Age:  99,
	}
	type args struct {
		ctx      context.Context
		desc     *Descriptor
		endpoint *Endpoint
	}
	tests := []struct {
		name    string
		args    args
		wantErr error
		errfn   func(*testing.T, error)
	}{{
		name: "call fails",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "\t\t\t\t", // causes failure
				Logger:  model.DiscardLogger,
			},
		},
		wantErr: errors.New(`parse "\t\t\t\t": net/url: invalid control character in URL`),
		errfn:   nil,
	}, {
		name: "with error during httpClient.Do",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: io.EOF,
		errfn: func(t *testing.T, err error) {
			var expect *errMaybeCensorship
			if !errors.As(err, &expect) {
				t.Fatal("invalid error type")
			}
		},
	}, {
		name: "with error when reading the response body",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body: io.NopCloser(&mocks.Reader{
								MockRead: func(b []byte) (int, error) {
									return 0, netxlite.ECONNRESET
								},
							}),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: errors.New(netxlite.FailureConnectionReset),
		errfn: func(t *testing.T, err error) {
			var expect *errMaybeCensorship
			if !errors.As(err, &expect) {
				t.Fatal("invalid error type")
			}
		},
	}, {
		name: "with HTTP failure",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader(`{"Name": "sbs", "Age": 99}`)),
							StatusCode: 400,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: errors.New("httpapi: http request failed: 400"),
		errfn: func(t *testing.T, err error) {
			var expect *ErrHTTPRequestFailed
			if !errors.As(err, &expect) {
				t.Fatal("invalid error type")
			}
		},
	}, {
		name: "with good response and missing header",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Body:       io.NopCloser(strings.NewReader(`{"Name": "sbs", "Age": 99}`)),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: nil,
		errfn:   nil,
	}, {
		name: "with good response and good header",
		args: args{
			ctx:  context.Background(),
			desc: &Descriptor{},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/a",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Header: http.Header{
								"Content-Type": {"application/json"},
							},
							Body:       io.NopCloser(strings.NewReader(`{"Name": "sbs", "Age": 99}`)),
							StatusCode: 200,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: nil,
		errfn:   nil,
	}, {
		name: "response is not JSON",
		args: args{
			ctx: context.Background(),
			desc: &Descriptor{
				LogBody: false,
				Method:  http.MethodGet,
			},
			endpoint: &Endpoint{
				BaseURL: "https://www.example.com/",
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						resp := &http.Response{
							Header: http.Header{
								"Content-Type": {"application/json"},
							},
							Body:       io.NopCloser(strings.NewReader(`{`)), // invalid JSON
							StatusCode: 200,
						}
						return resp, nil
					},
				},
				Logger: model.DiscardLogger,
			},
		},
		wantErr: errors.New("unexpected end of JSON input"),
		errfn:   nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var response response
			err := CallWithJSONResponse(tt.args.ctx, tt.args.desc, tt.args.endpoint, &response)
			switch {
			case err == nil && tt.wantErr == nil:
				if diff := cmp.Diff(expectedResponse, response); err != nil {
					t.Fatal(diff)
				}
			case err != nil && tt.wantErr == nil:
				t.Fatalf("expected <nil> error but got %s", err.Error())
			case err == nil && tt.wantErr != nil:
				t.Fatalf("expected %s but got <nil>", tt.wantErr.Error())
			case err.Error() == tt.wantErr.Error():
				// nothing
			default:
				t.Fatalf("expected %s but got %s", err.Error(), tt.wantErr.Error())
			}
			if tt.errfn != nil {
				tt.errfn(t, err)
			}
		})
	}
}

func TestCallHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // should fail HTTP request immediately
	desc := &Descriptor{
		LogBody: false,
		Method:  http.MethodGet,
		URLPath: "/robots.txt",
	}
	endpoint := &Endpoint{
		BaseURL:    "https://www.example.com/",
		HTTPClient: http.DefaultClient,
		Logger:     model.DiscardLogger,
		UserAgent:  model.HTTPHeaderUserAgent,
	}
	body, err := Call(ctx, desc, endpoint)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
	if len(body) > 0 {
		t.Fatal("expected zero-length body")
	}
}

func TestCallWithJSONResponseHonoursContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // should fail HTTP request immediately
	desc := &Descriptor{
		LogBody: false,
		Method:  http.MethodGet,
		URLPath: "/robots.txt",
	}
	endpoint := &Endpoint{
		BaseURL:    "https://www.example.com/",
		HTTPClient: http.DefaultClient,
		Logger:     model.DiscardLogger,
		UserAgent:  model.HTTPHeaderUserAgent,
	}
	var resp url.URL
	err := CallWithJSONResponse(ctx, desc, endpoint, &resp)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected err", err)
	}
}

func TestLogging(t *testing.T) {

	// This test was originally written for the httpx package and we have adapted it
	// by keeping the ~same implementation with a custom callx function that converts
	// the previous semantics of httpx to the new semantics of httpapi.
	callx := func(baseURL string, logBody bool, logger model.Logger, request, response any) error {
		desc := MustNewPOSTJSONWithJSONResponseDescriptor("/", request).WithBodyLogging(logBody)
		runtimex.Assert(desc.LogBody == logBody, "desc.LogBody should be equal to logBody here")
		endpoint := &Endpoint{
			BaseURL:    baseURL,
			HTTPClient: http.DefaultClient,
			Logger:     logger,
		}
		return CallWithJSONResponse(context.Background(), desc, endpoint, response)
	}

	// we also needed to create a constructor for the logger
	newlogger := func(logs chan string) model.Logger {
		return &mocks.Logger{
			MockDebugf: func(format string, v ...interface{}) {
				logs <- fmt.Sprintf(format, v...)
			},
			MockWarnf: func(format string, v ...interface{}) {
				logs <- fmt.Sprintf(format, v...)
			},
		}
	}

	t.Run("body logging enabled, 200 Ok, and without content-type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, true, logger, input, &output)
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				// we expect this because body logging is enabled
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				// we expect this because body logging is enabled
				found |= 1 << 1
				continue
			}
			if strings.HasPrefix(entry, "httpapi: unexpected content-type: ") {
				// we would expect this because the server does not send us any content-type
				found |= 1 << 2
				continue
			}
			if strings.HasPrefix(entry, "httpapi: request body length: ") {
				// we should see this because we sent a body
				found |= 1 << 3
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body length: ") {
				// we should see this because we receive a body
				found |= 1 << 4
				continue
			}
		}
		if found != (1<<0 | 1<<1 | 1<<2 | 1<<3 | 1<<4) {
			t.Fatal("did not find the expected logs")
		}
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("body logging enabled, 200 Ok, and with content-type", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Header().Add("content-type", "application/json")
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, true, logger, input, &output)
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				// we expect this because body logging is enabled
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				// we expect this because body logging is enabled
				found |= 1 << 1
				continue
			}
			if strings.HasPrefix(entry, "httpapi: unexpected content-type: ") {
				// we do not expect this because the server sends us a content-type
				found |= 1 << 2
				continue
			}
			if strings.HasPrefix(entry, "httpapi: request body length: ") {
				// we should see this because we sent a body
				found |= 1 << 3
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body length: ") {
				// we should see this because we receive a body
				found |= 1 << 4
				continue
			}
		}
		if found != (1<<0 | 1<<1 | 1<<3 | 1<<4) {
			t.Fatal("did not find the expected logs")
		}
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("body logging enabled and 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, true, logger, input, &output)
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				// should occur because body logging is enabled
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				// should occur because body logging is enabled
				found |= 1 << 1
				continue
			}
			if strings.HasPrefix(entry, "httpapi: unexpected content-type: ") {
				// note: this one should not occur because the code is 401 so we're not
				// actually going to parse the JSON document
				found |= 1 << 2
				continue
			}
			if strings.HasPrefix(entry, "httpapi: request body length: ") {
				// we should see this because we send a body
				found |= 1 << 3
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body length: ") {
				// we should see this because we receive a body
				found |= 1 << 4
				continue
			}
		}
		if found != (1<<0 | 1<<1 | 1<<3 | 1<<4) {
			t.Fatal("did not find the expected logs")
		}
		var failure *ErrHTTPRequestFailed
		if !errors.As(err, &failure) || failure.StatusCode != 401 {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("body logging NOT enabled and 200 Ok", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, false, logger, input, &output) // no logging
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				// should not see it: body logging is disabled
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				// should not see it: body logging is disabled
				found |= 1 << 1
				continue
			}
			if strings.HasPrefix(entry, "httpapi: unexpected content-type: ") {
				// this one should be logged ANYWAY because it's orthogonal to the
				// body logging so we should see it also in this case.
				found |= 1 << 2
				continue
			}
			if strings.HasPrefix(entry, "httpapi: request body length: ") {
				// should see this because we send a body
				found |= 1 << 3
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body length: ") {
				// should see this because we're receiving a body
				found |= 1 << 4
				continue
			}
		}
		if found != (1<<2 | 1<<3 | 1<<4) {
			t.Fatal("did not find the expected logs")
		}
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("body logging NOT enabled and 401 Unauthorized", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(
			func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				w.Write([]byte("[]"))
			},
		))
		logs := make(chan string, 1024)
		defer server.Close()
		var (
			input  []string
			output []string
		)
		logger := newlogger(logs)
		err := callx(server.URL, false, logger, input, &output) // no logging
		var found int
		close(logs)
		for entry := range logs {
			if strings.HasPrefix(entry, "httpapi: request body: ") {
				// should not see it: body logging is disabled
				found |= 1 << 0
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body: ") {
				// should not see it: body logging is disabled
				found |= 1 << 1
				continue
			}
			if strings.HasPrefix(entry, "httpapi: unexpected content-type: ") {
				// should not see it because we don't parse the body on 401 errors
				found |= 1 << 2
				continue
			}
			if strings.HasPrefix(entry, "httpapi: request body length: ") {
				// we send a body so we should see it
				found |= 1 << 3
				continue
			}
			if strings.HasPrefix(entry, "httpapi: response body length: ") {
				// we receive a body so we should see it
				found |= 1 << 4
				continue
			}
		}
		if found != (1<<3 | 1<<4) {
			t.Fatal("did not find the expected logs")
		}
		var failure *ErrHTTPRequestFailed
		if !errors.As(err, &failure) || failure.StatusCode != 401 {
			t.Fatal("unexpected err", err)
		}
	})
}

func Test_errMaybeCensorship_Unwrap(t *testing.T) {
	t.Run("for errors.Is", func(t *testing.T) {
		var err error = &errMaybeCensorship{io.EOF}
		if !errors.Is(err, io.EOF) {
			t.Fatal("cannot unwrap")
		}
	})

	t.Run("for errors.As", func(t *testing.T) {
		var err error = &errMaybeCensorship{netxlite.ECONNRESET}
		var syserr syscall.Errno
		if !errors.As(err, &syserr) || syserr != netxlite.ECONNRESET {
			t.Fatal("cannot unwrap")
		}
	})
}
