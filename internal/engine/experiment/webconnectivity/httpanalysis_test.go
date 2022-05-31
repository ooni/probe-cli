package webconnectivity_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tracex"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

func TestHTTPBodyLengthChecks(t *testing.T) {
	var (
		trueValue  = true
		falseValue = false
	)
	type args struct {
		tk   urlgetter.TestKeys
		ctrl webconnectivity.ControlResponse
	}
	tests := []struct {
		name        string
		args        args
		lengthMatch *bool
		proportion  float64
	}{{
		name:        "nothing",
		args:        args{},
		lengthMatch: nil,
	}, {
		name: "control length is nonzero",
		args: args{
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 1024,
				},
			},
		},
		lengthMatch: nil,
	}, {
		name: "response body is truncated",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						BodyIsTruncated: true,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 1024,
				},
			},
		},
		lengthMatch: nil,
	}, {
		name: "response body length is zero",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 1024,
				},
			},
		},
		lengthMatch: nil,
	}, {
		name: "control length is negative",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Body: tracex.MaybeBinaryValue{
							Value: randx.Letters(768),
						},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: -1,
				},
			},
		},
		lengthMatch: nil,
	}, {
		name: "match with bigger control",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Body: tracex.MaybeBinaryValue{
							Value: randx.Letters(768),
						},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 1024,
				},
			},
		},
		lengthMatch: &trueValue,
		proportion:  0.75,
	}, {
		name: "match with bigger measurement",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Body: tracex.MaybeBinaryValue{
							Value: randx.Letters(1024),
						},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 768,
				},
			},
		},
		lengthMatch: &trueValue,
		proportion:  0.75,
	}, {
		name: "not match with bigger control",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Body: tracex.MaybeBinaryValue{
							Value: randx.Letters(8),
						},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 16,
				},
			},
		},
		lengthMatch: &falseValue,
		proportion:  0.5,
	}, {
		name: "match with bigger measurement",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Body: tracex.MaybeBinaryValue{
							Value: randx.Letters(16),
						},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					BodyLength: 8,
				},
			},
		},
		lengthMatch: &falseValue,
		proportion:  0.5,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			match, proportion := webconnectivity.HTTPBodyLengthChecks(tt.args.tk, tt.args.ctrl)
			if diff := cmp.Diff(tt.lengthMatch, match); diff != "" {
				t.Fatal(diff)
			}
			if diff := cmp.Diff(tt.proportion, proportion); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestStatusCodeMatch(t *testing.T) {
	var (
		trueValue  = true
		falseValue = false
	)
	type args struct {
		tk   urlgetter.TestKeys
		ctrl webconnectivity.ControlResponse
	}
	tests := []struct {
		name    string
		args    args
		wantOut *bool
	}{{
		name: "with all zero",
		args: args{},
	}, {
		name: "with a request but zero status codes",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{}},
			},
		},
	}, {
		name: "with equal status codes including 5xx",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 501,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 501,
				},
			},
		},
		wantOut: &trueValue,
	}, {
		name: "with different status codes and the control being 5xx",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 407,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 501,
				},
			},
		},
		wantOut: nil,
	}, {
		name: "with different status codes and the control being not 5xx",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 407,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
				},
			},
		},
		wantOut: &falseValue,
	}, {
		name: "with only response status code and no control status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
					},
				}},
			},
		},
	}, {
		name: "with response status code and -1 as control status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: -1,
				},
			},
		},
	}, {
		name: "with only control status code and no response status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 0,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
				},
			},
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.HTTPStatusCodeMatch(tt.args.tk, tt.args.ctrl)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestHeadersMatch(t *testing.T) {
	var (
		trueValue  = true
		falseValue = false
	)
	type args struct {
		tk   urlgetter.TestKeys
		ctrl webconnectivity.ControlResponse
	}
	tests := []struct {
		name string
		args args
		want *bool
	}{{
		name: "with no requests",
		args: args{
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Date":   "Mon Jul 13 21:05:43 CEST 2020",
						"Antani": "Mascetti",
					},
					StatusCode: 200,
				},
			},
		},
		want: nil,
	}, {
		name: "with basically nothing",
		args: args{},
		want: nil,
	}, {
		name: "with request and no response status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Date":   "Mon Jul 13 21:05:43 CEST 2020",
						"Antani": "Mascetti",
					},
					StatusCode: 200,
				},
			},
		},
		want: nil,
	}, {
		name: "with no control status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Date": {Value: "Mon Jul 13 21:10:08 CEST 2020"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{},
		},
		want: nil,
	}, {
		name: "with negative control status code",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Date": {Value: "Mon Jul 13 21:10:08 CEST 2020"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: -1,
				},
			},
		},
		want: nil,
	}, {
		name: "with no uncommon headers",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Date": {Value: "Mon Jul 13 21:10:08 CEST 2020"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Date": "Mon Jul 13 21:05:43 CEST 2020",
					},
					StatusCode: 200,
				},
			},
		},
		want: &trueValue,
	}, {
		name: "with equal uncommon headers",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Date":   {Value: "Mon Jul 13 21:10:08 CEST 2020"},
							"Antani": {Value: "MASCETTI"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Date":   "Mon Jul 13 21:05:43 CEST 2020",
						"Antani": "MELANDRI",
					},
					StatusCode: 200,
				},
			},
		},
		want: &trueValue,
	}, {
		name: "with different uncommon headers",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Date":   {Value: "Mon Jul 13 21:10:08 CEST 2020"},
							"Antani": {Value: "MASCETTI"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Date":     "Mon Jul 13 21:05:43 CEST 2020",
						"Melandri": "MASCETTI",
					},
					StatusCode: 200,
				},
			},
		},
		want: &falseValue,
	}, {
		name: "with small uncommon intersection (X-Cache)",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Accept-Ranges":  {Value: "bytes"},
							"Age":            {Value: "404727"},
							"Cache-Control":  {Value: "max-age=604800"},
							"Content-Length": {Value: "1256"},
							"Content-Type":   {Value: "text/html; charset=UTF-8"},
							"Date":           {Value: "Tue, 14 Jul 2020 22:26:09 GMT"},
							"Etag":           {Value: "\"3147526947\""},
							"Expires":        {Value: "Tue, 21 Jul 2020 22:26:09 GMT"},
							"Last-Modified":  {Value: "Thu, 17 Oct 2019 07:18:26 GMT"},
							"Server":         {Value: "ECS (dcb/7F3C)"},
							"Vary":           {Value: "Accept-Encoding"},
							"X-Cache":        {Value: "HIT"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						// Note: the test helper was probably requesting the
						// resource in a different way. There is content-length
						// in this response, maybe it's using HTTP/1.0?
						"Accept-Ranges": "bytes",
						"Age":           "469182",
						"Cache-Control": "max-age=604800",
						"Content-Type":  "text/html; charset=UTF-8",
						"Date":          "Tue, 14 Jul 2020 22:26:08 GMT",
						"Etag":          "\"3147526947\"",
						"Expires":       "Tue, 21 Jul 2020 22:26:08 GMT",
						"Last-Modified": "Thu, 17 Oct 2019 07:18:26 GMT",
						"Server":        "ECS (nyb/1D07)",
						"Vary":          "Accept-Encoding",
						"X-Cache":       "HIT",
					},
					StatusCode: 200,
				},
			},
		},
		want: &trueValue,
	}, {
		name: "with no uncommon intersection",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Accept-Ranges":  {Value: "bytes"},
							"Age":            {Value: "404727"},
							"Cache-Control":  {Value: "max-age=604800"},
							"Content-Length": {Value: "1256"},
							"Content-Type":   {Value: "text/html; charset=UTF-8"},
							"Date":           {Value: "Tue, 14 Jul 2020 22:26:09 GMT"},
							"Etag":           {Value: "\"3147526947\""},
							"Expires":        {Value: "Tue, 21 Jul 2020 22:26:09 GMT"},
							"Last-Modified":  {Value: "Thu, 17 Oct 2019 07:18:26 GMT"},
							"Server":         {Value: "ECS (dcb/7F3C)"},
							"Vary":           {Value: "Accept-Encoding"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						// Note: the test helper was probably requesting the
						// resource in a different way. There is content-length
						// in this response, maybe it's using HTTP/1.0?
						"Accept-Ranges": "bytes",
						"Age":           "469182",
						"Cache-Control": "max-age=604800",
						"Content-Type":  "text/html; charset=UTF-8",
						"Date":          "Tue, 14 Jul 2020 22:26:08 GMT",
						"Etag":          "\"3147526947\"",
						"Expires":       "Tue, 21 Jul 2020 22:26:08 GMT",
						"Last-Modified": "Thu, 17 Oct 2019 07:18:26 GMT",
						"Server":        "ECS (nyb/1D07)",
						"Vary":          "Accept-Encoding",
					},
					StatusCode: 200,
				},
			},
		},
		want: &falseValue,
	}, {
		name: "with exactly equal headers",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"Accept-Ranges": {Value: "bytes"},
							"Age":           {Value: "404727"},
							"Cache-Control": {Value: "max-age=604800"},
							"Content-Type":  {Value: "text/html; charset=UTF-8"},
							"Date":          {Value: "Tue, 14 Jul 2020 22:26:09 GMT"},
							"Etag":          {Value: "\"3147526947\""},
							"Expires":       {Value: "Tue, 21 Jul 2020 22:26:09 GMT"},
							"Last-Modified": {Value: "Thu, 17 Oct 2019 07:18:26 GMT"},
							"Server":        {Value: "ECS (dcb/7F3C)"},
							"Vary":          {Value: "Accept-Encoding"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Accept-Ranges": "bytes",
						"Age":           "469182",
						"Cache-Control": "max-age=604800",
						"Content-Type":  "text/html; charset=UTF-8",
						"Date":          "Tue, 14 Jul 2020 22:26:08 GMT",
						"Etag":          "\"3147526947\"",
						"Expires":       "Tue, 21 Jul 2020 22:26:08 GMT",
						"Last-Modified": "Thu, 17 Oct 2019 07:18:26 GMT",
						"Server":        "ECS (nyb/1D07)",
						"Vary":          "Accept-Encoding",
					},
					StatusCode: 200,
				},
			},
		},
		want: &trueValue,
	}, {
		name: "with equal headers except for the case",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Headers: map[string]tracex.MaybeBinaryValue{
							"accept-ranges": {Value: "bytes"},
							"AGE":           {Value: "404727"},
							"cache-Control": {Value: "max-age=604800"},
							"Content-TyPe":  {Value: "text/html; charset=UTF-8"},
							"DatE":          {Value: "Tue, 14 Jul 2020 22:26:09 GMT"},
							"etag":          {Value: "\"3147526947\""},
							"expires":       {Value: "Tue, 21 Jul 2020 22:26:09 GMT"},
							"Last-Modified": {Value: "Thu, 17 Oct 2019 07:18:26 GMT"},
							"SerVer":        {Value: "ECS (dcb/7F3C)"},
							"Vary":          {Value: "Accept-Encoding"},
						},
						Code: 200,
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					Headers: map[string]string{
						"Accept-Ranges": "bytes",
						"Age":           "469182",
						"Cache-Control": "max-age=604800",
						"Content-Type":  "text/html; charset=UTF-8",
						"Date":          "Tue, 14 Jul 2020 22:26:08 GMT",
						"Etag":          "\"3147526947\"",
						"Expires":       "Tue, 21 Jul 2020 22:26:08 GMT",
						"Last-Modified": "Thu, 17 Oct 2019 07:18:26 GMT",
						"Server":        "ECS (nyb/1D07)",
						"Vary":          "Accept-Encoding",
					},
					StatusCode: 200,
				},
			},
		},
		want: &trueValue,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := webconnectivity.HTTPHeadersMatch(tt.args.tk, tt.args.ctrl)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTitleMatch(t *testing.T) {
	var (
		trueValue  = true
		falseValue = false
	)
	type args struct {
		tk   urlgetter.TestKeys
		ctrl webconnectivity.ControlResponse
	}
	tests := []struct {
		name    string
		args    args
		wantOut *bool
	}{{
		name:    "with all empty",
		args:    args{},
		wantOut: nil,
	}, {
		name: "with a request and no response",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{}},
			},
		},
		wantOut: nil,
	}, {
		name: "with a response with truncated body",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code:            200,
						BodyIsTruncated: true,
					},
				}},
			},
		},
		wantOut: nil,
	}, {
		name: "with a response with good body",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{Value: "<HTML/>"},
					},
				}},
			},
		},
		wantOut: nil,
	}, {
		name: "with all good but no titles",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{Value: "<HTML/>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
					Title:      "",
				},
			},
		},
		wantOut: nil,
	}, {
		name: "reasonably common case where it succeeds",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{
							Value: "<HTML><TITLE>La community di MSN</TITLE></HTML>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
					Title:      "MSN Community",
				},
			},
		},
		wantOut: &trueValue,
	}, {
		name: "reasonably common case where it fails",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{
							Value: "<HTML><TITLE>La communità di MSN</TITLE></HTML>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
					Title:      "MSN Community",
				},
			},
		},
		wantOut: &falseValue,
	}, {
		name: "when the title is too long",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{
							Value: "<HTML><TITLE>" + randx.Letters(1024) + "</TITLE></HTML>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
					Title:      "MSN Community",
				},
			},
		},
		wantOut: nil,
	}, {
		name: "reasonably common case where it succeeds with case variations",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{
							Value: "<HTML><TiTLe>La commUNity di MSN</tITLE></HTML>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: 200,
					Title:      "MSN COmmunity",
				},
			},
		},
		wantOut: &trueValue,
	}, {
		name: "when the control status code is negative",
		args: args{
			tk: urlgetter.TestKeys{
				Requests: []tracex.RequestEntry{{
					Response: tracex.HTTPResponse{
						Code: 200,
						Body: tracex.MaybeBinaryValue{
							Value: "<HTML><TiTLe>La commUNity di MSN</tITLE></HTML>"},
					},
				}},
			},
			ctrl: webconnectivity.ControlResponse{
				HTTPRequest: webconnectivity.ControlHTTPRequestResult{
					StatusCode: -1,
				},
			},
		},
		wantOut: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.HTTPTitleMatch(tt.args.tk, tt.args.ctrl)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
