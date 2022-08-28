package measurexlite

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
)

func TestNewArchivalHTTPRequestResult(t *testing.T) {
	type args struct {
		index           int64
		started         time.Duration
		network         string
		address         string
		alpn            string
		transport       string
		req             *http.Request
		resp            *http.Response
		maxRespBodySize int64
		body            []byte
		err             error
		finished        time.Duration
	}

	type config struct {
		name   string
		args   args
		expect *model.ArchivalHTTPRequestResult
	}

	configs := []config{{
		name: "the code is defensive with all zero-value inputs",
		args: args{
			index:           0,
			started:         0,
			network:         "",
			address:         "",
			alpn:            "",
			transport:       "",
			req:             nil,
			resp:            nil,
			maxRespBodySize: 0,
			body:            nil,
			err:             nil,
			finished:        0,
		},
		expect: &model.ArchivalHTTPRequestResult{
			Network: "",
			Address: "",
			ALPN:    "",
			Failure: nil,
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList:     []model.ArchivalHTTPHeader{},
				Headers:         map[string]model.ArchivalMaybeBinaryData{},
				Method:          "",
				Tor:             model.ArchivalHTTPTor{},
				Transport:       "",
				URL:             "",
			},
			Response: model.ArchivalHTTPResponse{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				Code:            0,
				HeadersList:     []model.ArchivalHTTPHeader{},
				Headers:         map[string]model.ArchivalMaybeBinaryData{},
				Locations:       []string{},
			},
			T0:            0,
			T:             0,
			TransactionID: 0,
		},
	}, {
		name: "case of request that failed with I/O issues",
		args: args{
			index:     1,
			started:   250 * time.Millisecond,
			network:   "tcp",
			address:   "8.8.8.8:80",
			alpn:      "",
			transport: "tcp",
			req: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "http",
					Host:   "dns.google",
					Path:   "/",
				},
				Header: http.Header{
					"Accept":     {"*/*"},
					"User-Agent": {"miniooni/0.1.0-dev"},
				},
			},
			resp:            nil,
			maxRespBodySize: 1 << 19,
			body:            nil,
			err:             netxlite.NewTopLevelGenericErrWrapper(netxlite.ECONNRESET),
			finished:        750 * time.Millisecond,
		},
		expect: &model.ArchivalHTTPRequestResult{
			Network: "tcp",
			Address: "8.8.8.8:80",
			ALPN:    "",
			Failure: func() *string {
				s := netxlite.FailureConnectionReset
				return &s
			}(),
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Accept",
					Value: model.ArchivalMaybeBinaryData{
						Value: "*/*",
					},
				}, {
					Key: "User-Agent",
					Value: model.ArchivalMaybeBinaryData{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Accept":     {Value: "*/*"},
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method:    "GET",
				Tor:       model.ArchivalHTTPTor{},
				Transport: "tcp",
				URL:       "http://dns.google/",
			},
			Response: model.ArchivalHTTPResponse{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				Code:            0,
				HeadersList:     []model.ArchivalHTTPHeader{},
				Headers:         map[string]model.ArchivalMaybeBinaryData{},
				Locations:       []string{},
			},
			T0:            0.25,
			T:             0.75,
			TransactionID: 1,
		},
	}, {
		name: "case of request that succeded",
		args: args{
			index:     44,
			started:   1400 * time.Millisecond,
			network:   "udp",
			address:   "8.8.8.8:443",
			alpn:      "h3",
			transport: "quic",
			req: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "dns.google",
					Path:   "/",
				},
				Header: http.Header{
					"Accept":     {"*/*"},
					"User-Agent": {"miniooni/0.1.0-dev"},
				},
			},
			resp: &http.Response{
				StatusCode: 200,
				Header: http.Header{
					"Content-Type": {"text/html; charset=iso-8859-1"},
					"Server":       {"Apache"},
				},
			},
			maxRespBodySize: 1 << 19,
			body:            filtering.HTTPBlockpage451,
			err:             nil,
			finished:        1500 * time.Millisecond,
		},
		expect: &model.ArchivalHTTPRequestResult{
			Network: "udp",
			Address: "8.8.8.8:443",
			ALPN:    "h3",
			Failure: nil,
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Accept",
					Value: model.ArchivalMaybeBinaryData{
						Value: "*/*",
					},
				}, {
					Key: "User-Agent",
					Value: model.ArchivalMaybeBinaryData{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Accept":     {Value: "*/*"},
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method:    "GET",
				Tor:       model.ArchivalHTTPTor{},
				Transport: "quic",
				URL:       "https://dns.google/",
			},
			Response: model.ArchivalHTTPResponse{
				Body: model.ArchivalMaybeBinaryData{
					Value: string(filtering.HTTPBlockpage451),
				},
				BodyIsTruncated: false,
				Code:            200,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Content-Type",
					Value: model.ArchivalMaybeBinaryData{
						Value: "text/html; charset=iso-8859-1",
					},
				}, {
					Key: "Server",
					Value: model.ArchivalMaybeBinaryData{
						Value: "Apache",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Content-Type": {Value: "text/html; charset=iso-8859-1"},
					"Server":       {Value: "Apache"},
				},
				Locations: []string{},
			},
			T0:            1.4,
			T:             1.5,
			TransactionID: 44,
		},
	}, {
		name: "case of redirect",
		args: args{
			index:     47,
			started:   1400 * time.Millisecond,
			network:   "udp",
			address:   "8.8.8.8:443",
			alpn:      "h3",
			transport: "quic",
			req: &http.Request{
				Method: "GET",
				URL: &url.URL{
					Scheme: "https",
					Host:   "dns.google",
					Path:   "/",
				},
				Header: http.Header{
					"Accept":     {"*/*"},
					"User-Agent": {"miniooni/0.1.0-dev"},
				},
			},
			resp: &http.Response{
				StatusCode: 302,
				Header: http.Header{
					"Content-Type": {"text/html; charset=iso-8859-1"},
					"Location":     {"/v2/index.html"},
					"Server":       {"Apache"},
				},
				Request: &http.Request{ // necessary for Location to WAI
					URL: &url.URL{
						Scheme: "https",
						Host:   "dns.google",
						Path:   "/",
					},
				},
			},
			maxRespBodySize: 1 << 19,
			body:            nil,
			err:             nil,
			finished:        1500 * time.Millisecond,
		},
		expect: &model.ArchivalHTTPRequestResult{
			Network: "udp",
			Address: "8.8.8.8:443",
			ALPN:    "h3",
			Failure: nil,
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Accept",
					Value: model.ArchivalMaybeBinaryData{
						Value: "*/*",
					},
				}, {
					Key: "User-Agent",
					Value: model.ArchivalMaybeBinaryData{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Accept":     {Value: "*/*"},
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method:    "GET",
				Tor:       model.ArchivalHTTPTor{},
				Transport: "quic",
				URL:       "https://dns.google/",
			},
			Response: model.ArchivalHTTPResponse{
				Body: model.ArchivalMaybeBinaryData{
					Value: "",
				},
				BodyIsTruncated: false,
				Code:            302,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Content-Type",
					Value: model.ArchivalMaybeBinaryData{
						Value: "text/html; charset=iso-8859-1",
					},
				}, {
					Key: "Location",
					Value: model.ArchivalMaybeBinaryData{
						Value: "/v2/index.html",
					},
				}, {
					Key: "Server",
					Value: model.ArchivalMaybeBinaryData{
						Value: "Apache",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Content-Type": {Value: "text/html; charset=iso-8859-1"},
					"Location":     {Value: "/v2/index.html"},
					"Server":       {Value: "Apache"},
				},
				Locations: []string{
					"https://dns.google/v2/index.html",
				},
			},
			T0:            1.4,
			T:             1.5,
			TransactionID: 47,
		},
	}}

	for _, cnf := range configs {
		t.Run(cnf.name, func(t *testing.T) {
			out := NewArchivalHTTPRequestResult(
				cnf.args.index,
				cnf.args.started,
				cnf.args.network,
				cnf.args.address,
				cnf.args.alpn,
				cnf.args.transport,
				cnf.args.req,
				cnf.args.resp,
				cnf.args.maxRespBodySize,
				cnf.args.body,
				cnf.args.err,
				cnf.args.finished,
			)
			if diff := cmp.Diff(cnf.expect, out); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
