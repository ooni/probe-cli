package oonidatamodel

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/oonitemplates"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

func TestNewTCPConnectListEmpty(t *testing.T) {
	out := NewTCPConnectList(oonitemplates.Results{})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewTCPConnectListSuccess(t *testing.T) {
	out := NewTCPConnectList(oonitemplates.Results{
		Connects: []*modelx.ConnectEvent{
			{
				RemoteAddress: "8.8.8.8:53",
			},
			{
				RemoteAddress: "8.8.4.4:853",
			},
		},
	})
	if len(out) != 2 {
		t.Fatal("unexpected output length")
	}
	if out[0].IP != "8.8.8.8" {
		t.Fatal("unexpected out[0].IP")
	}
	if out[0].Port != 53 {
		t.Fatal("unexpected out[0].Port")
	}
	if out[0].Status.Failure != nil {
		t.Fatal("unexpected out[0].Failure")
	}
	if out[0].Status.Success != true {
		t.Fatal("unexpected out[0].Success")
	}
	if out[1].IP != "8.8.4.4" {
		t.Fatal("unexpected out[1].IP")
	}
	if out[1].Port != 853 {
		t.Fatal("unexpected out[1].Port")
	}
	if out[1].Status.Failure != nil {
		t.Fatal("unexpected out[0].Failure")
	}
	if out[1].Status.Success != true {
		t.Fatal("unexpected out[0].Success")
	}
}

func TestNewTCPConnectListFailure(t *testing.T) {
	out := NewTCPConnectList(oonitemplates.Results{
		Connects: []*modelx.ConnectEvent{
			{
				RemoteAddress: "8.8.8.8:53",
				Error:         errors.New(errorx.FailureConnectionReset),
			},
		},
	})
	if len(out) != 1 {
		t.Fatal("unexpected output length")
	}
	if out[0].IP != "8.8.8.8" {
		t.Fatal("unexpected out[0].IP")
	}
	if out[0].Port != 53 {
		t.Fatal("unexpected out[0].Port")
	}
	if *out[0].Status.Failure != errorx.FailureConnectionReset {
		t.Fatal("unexpected out[0].Failure")
	}
	if out[0].Status.Success != false {
		t.Fatal("unexpected out[0].Success")
	}
}

func TestNewTCPConnectListInvalidInput(t *testing.T) {
	out := NewTCPConnectList(oonitemplates.Results{
		Connects: []*modelx.ConnectEvent{
			{
				RemoteAddress: "8.8.8.8",
				Error:         errors.New(errorx.FailureConnectionReset),
			},
		},
	})
	if len(out) != 1 {
		t.Fatal("unexpected output length")
	}
	if out[0].IP != "" {
		t.Fatal("unexpected out[0].IP")
	}
	if out[0].Port != 0 {
		t.Fatal("unexpected out[0].Port")
	}
	if *out[0].Status.Failure != errorx.FailureConnectionReset {
		t.Fatal("unexpected out[0].Failure")
	}
	if out[0].Status.Success != false {
		t.Fatal("unexpected out[0].Success")
	}
}

func TestNewRequestsListEmptyList(t *testing.T) {
	out := NewRequestList(oonitemplates.Results{})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewRequestsListGood(t *testing.T) {
	out := NewRequestList(oonitemplates.Results{
		HTTPRequests: []*modelx.HTTPRoundTripDoneEvent{
			// need two requests to test that order is inverted
			{
				RequestBodySnap: []byte("abcdefx"),
				RequestHeaders: http.Header{
					"Content-Type": []string{
						"text/plain",
						"foobar",
					},
					"Content-Length": []string{
						"17",
					},
				},
				RequestMethod:    "GET",
				RequestURL:       "http://x.org/",
				ResponseBodySnap: []byte("abcdef"),
				ResponseHeaders: http.Header{
					"Content-Type": []string{
						"application/json",
						"foobaz",
					},
					"Server": []string{
						"antani",
					},
					"Content-Length": []string{
						"14",
					},
				},
				ResponseStatusCode: 451,
				MaxBodySnapSize:    10,
			},
			{
				Error: errors.New("antani"),
			},
		},
	})
	if len(out) != 2 {
		t.Fatal("unexpected output length")
	}

	if *out[0].Failure != "antani" {
		t.Fatal("unexpected out[0].Failure")
	}
	if out[0].Request.Body.Value != "" {
		t.Fatal("unexpected out[0].Request.Body.Value")
	}
	if len(out[0].Request.Headers) != 0 {
		t.Fatal("unexpected out[0].Request.Headers")
	}
	if out[0].Request.Method != "" {
		t.Fatal("unexpected out[0].Request.Method")
	}
	if out[0].Request.URL != "" {
		t.Fatal("unexpected out[0].Request.URL")
	}
	if out[0].Request.BodyIsTruncated != false {
		t.Fatal("unexpected out[0].Request.BodyIsTruncated")
	}
	if out[0].Response.Body.Value != "" {
		t.Fatal("unexpected out[0].Response.Body.Value")
	}
	if out[0].Response.Code != 0 {
		t.Fatal("unexpected out[0].Response.Code")
	}
	if len(out[0].Response.Headers) != 0 {
		t.Fatal("unexpected out[0].Response.Headers")
	}
	if out[0].Response.BodyIsTruncated != false {
		t.Fatal("unexpected out[0].Response.BodyIsTruncated")
	}

	if out[1].Failure != nil {
		t.Fatal("unexpected out[1].Failure")
	}
	if out[1].Request.Body.Value != "abcdefx" {
		t.Fatal("unexpected out[1].Request.Body.Value")
	}
	if len(out[1].Request.Headers) != 2 {
		t.Fatal("unexpected out[1].Request.Headers")
	}
	if out[1].Request.Headers["Content-Type"].Value != "text/plain" {
		t.Fatal("unexpected out[1].Request.Headers Content-Type value")
	}
	if out[1].Request.Headers["Content-Length"].Value != "17" {
		t.Fatal("unexpected out[1].Request.Headers Content-Length value")
	}
	var (
		requestHasTextPlain     bool
		requestHasFoobar        bool
		requestHasContentLength bool
		requestHasOther         int64
	)
	for _, header := range out[1].Request.HeadersList {
		if header.Key == "Content-Type" {
			if header.Value.Value == "text/plain" {
				requestHasTextPlain = true
			} else if header.Value.Value == "foobar" {
				requestHasFoobar = true
			} else {
				requestHasOther++
			}
		} else if header.Key == "Content-Length" {
			if header.Value.Value == "17" {
				requestHasContentLength = true
			} else {
				requestHasOther++
			}
		} else {
			requestHasOther++
		}
	}
	if !requestHasTextPlain {
		t.Fatal("missing text/plain for request")
	}
	if !requestHasFoobar {
		t.Fatal("missing foobar for request")
	}
	if !requestHasContentLength {
		t.Fatal("missing content_length for request")
	}
	if requestHasOther != 0 {
		t.Fatal("seen something unexpected")
	}
	if out[1].Request.Method != "GET" {
		t.Fatal("unexpected out[1].Request.Method")
	}
	if out[1].Request.URL != "http://x.org/" {
		t.Fatal("unexpected out[1].Request.URL")
	}
	if out[1].Request.BodyIsTruncated != false {
		t.Fatal("unexpected out[1].Request.BodyIsTruncated")
	}

	if out[1].Response.Body.Value != "abcdef" {
		t.Fatal("unexpected out[1].Response.Body.Value")
	}
	if out[1].Response.Code != 451 {
		t.Fatal("unexpected out[1].Response.Code")
	}
	if len(out[1].Response.Headers) != 3 {
		t.Fatal("unexpected out[1].Response.Headers")
	}
	if out[1].Response.Headers["Content-Type"].Value != "application/json" {
		t.Fatal("unexpected out[1].Response.Headers Content-Type value")
	}
	if out[1].Response.Headers["Server"].Value != "antani" {
		t.Fatal("unexpected out[1].Response.Headers Server value")
	}
	if out[1].Response.Headers["Content-Length"].Value != "14" {
		t.Fatal("unexpected out[1].Response.Headers Content-Length value")
	}
	var (
		responseHasApplicationJSON bool
		responseHasFoobaz          bool
		responseHasServer          bool
		responseHasContentLength   bool
		responseHasOther           int64
	)
	for _, header := range out[1].Response.HeadersList {
		if header.Key == "Content-Type" {
			if header.Value.Value == "application/json" {
				responseHasApplicationJSON = true
			} else if header.Value.Value == "foobaz" {
				responseHasFoobaz = true
			} else {
				responseHasOther++
			}
		} else if header.Key == "Content-Length" {
			if header.Value.Value == "14" {
				responseHasContentLength = true
			} else {
				responseHasOther++
			}
		} else if header.Key == "Server" {
			if header.Value.Value == "antani" {
				responseHasServer = true
			} else {
				responseHasOther++
			}
		} else {
			responseHasOther++
		}
	}
	if !responseHasApplicationJSON {
		t.Fatal("missing application/json for response")
	}
	if !responseHasFoobaz {
		t.Fatal("missing foobaz for response")
	}
	if !responseHasContentLength {
		t.Fatal("missing content_length for response")
	}
	if !responseHasServer {
		t.Fatal("missing server for response")
	}
	if responseHasOther != 0 {
		t.Fatal("seen something unexpected")
	}
	if out[1].Response.BodyIsTruncated != false {
		t.Fatal("unexpected out[1].Response.BodyIsTruncated")
	}
}

func TestNewRequestsSnaps(t *testing.T) {
	out := NewRequestList(oonitemplates.Results{
		HTTPRequests: []*modelx.HTTPRoundTripDoneEvent{
			{
				RequestBodySnap:  []byte("abcd"),
				MaxBodySnapSize:  4,
				ResponseBodySnap: []byte("defg"),
			},
		},
	})
	if len(out) != 1 {
		t.Fatal("unexpected output length")
	}
	if out[0].Request.BodyIsTruncated != true {
		t.Fatal("wrong out[0].Request.BodyIsTruncated")
	}
	if out[0].Response.BodyIsTruncated != true {
		t.Fatal("wrong out[0].Response.BodyIsTruncated")
	}
}

func TestMarshalUnmarshalHTTPBodyString(t *testing.T) {
	mbv := HTTPBody{
		Value: "1234",
	}
	data, err := json.Marshal(mbv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte(`"1234"`)) {
		t.Fatal("result is unexpected")
	}
	var newbody HTTPBody
	if err := json.Unmarshal(data, &newbody); err != nil {
		t.Fatal(err)
	}
	if newbody.Value != mbv.Value {
		t.Fatal("string value mistmatch")
	}
}

var binaryInput = []uint8{
	0x57, 0xe5, 0x79, 0xfb, 0xa6, 0xbb, 0x0d, 0xbc, 0xce, 0xbd, 0xa7, 0xa0,
	0xba, 0xa4, 0x78, 0x78, 0x12, 0x59, 0xee, 0x68, 0x39, 0xa4, 0x07, 0x98,
	0xc5, 0x3e, 0xbc, 0x55, 0xcb, 0xfe, 0x34, 0x3c, 0x7e, 0x1b, 0x5a, 0xb3,
	0x22, 0x9d, 0xc1, 0x2d, 0x6e, 0xca, 0x5b, 0xf1, 0x10, 0x25, 0x47, 0x1e,
	0x44, 0xe2, 0x2d, 0x60, 0x08, 0xea, 0xb0, 0x0a, 0xcc, 0x05, 0x48, 0xa0,
	0xf5, 0x78, 0x38, 0xf0, 0xdb, 0x3f, 0x9d, 0x9f, 0x25, 0x6f, 0x89, 0x00,
	0x96, 0x93, 0xaf, 0x43, 0xac, 0x4d, 0xc9, 0xac, 0x13, 0xdb, 0x22, 0xbe,
	0x7a, 0x7d, 0xd9, 0x24, 0xa2, 0x52, 0x69, 0xd8, 0x89, 0xc1, 0xd1, 0x57,
	0xaa, 0x04, 0x2b, 0xa2, 0xd8, 0xb1, 0x19, 0xf6, 0xd5, 0x11, 0x39, 0xbb,
	0x80, 0xcf, 0x86, 0xf9, 0x5f, 0x9d, 0x8c, 0xab, 0xf5, 0xc5, 0x74, 0x24,
	0x3a, 0xa2, 0xd4, 0x40, 0x4e, 0xd7, 0x10, 0x1f,
}

func TestMarshalUnmarshalHTTPBodyBinary(t *testing.T) {
	mbv := HTTPBody{
		Value: string(binaryInput),
	}
	data, err := json.Marshal(mbv)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(data, []byte(`{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}`)) {
		t.Fatal("result is unexpected")
	}
	var newbody HTTPBody
	if err := json.Unmarshal(data, &newbody); err != nil {
		t.Fatal(err)
	}
	if newbody.Value != mbv.Value {
		t.Fatal("string value mistmatch")
	}
}

func TestMaybeBinaryValueUnmarshalJSON(t *testing.T) {
	t.Run("when the code is not a map or string", func(t *testing.T) {
		var (
			mbv   MaybeBinaryValue
			input = []byte("[1, 2, 3, 4]")
		)
		if err := json.Unmarshal(input, &mbv); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the format field is missing", func(t *testing.T) {
		var (
			mbv   MaybeBinaryValue
			input = []byte("{}")
		)
		if err := json.Unmarshal(input, &mbv); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the format field is invalid", func(t *testing.T) {
		var (
			mbv   MaybeBinaryValue
			input = []byte(`{"format":"antani"}`)
		)
		if err := json.Unmarshal(input, &mbv); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is missing", func(t *testing.T) {
		var (
			mbv   MaybeBinaryValue
			input = []byte(`{"format":"base64"}`)
		)
		if err := json.Unmarshal(input, &mbv); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is not base64", func(t *testing.T) {
		var (
			mbv   MaybeBinaryValue
			input = []byte(`{"format":"base64","data":"antani"}`)
		)
		if err := json.Unmarshal(input, &mbv); err == nil {
			t.Fatal("expected an error here")
		}
	})
}

func TestMarshalUnmarshalHTTPHeaderString(t *testing.T) {
	mbh := HTTPHeadersList{
		HTTPHeader{
			Key: "Content-Type",
			Value: MaybeBinaryValue{
				Value: "application/json",
			},
		},
		HTTPHeader{
			Key: "Content-Type",
			Value: MaybeBinaryValue{
				Value: "antani",
			},
		},
		HTTPHeader{
			Key: "Content-Length",
			Value: MaybeBinaryValue{
				Value: "17",
			},
		},
	}
	data, err := json.Marshal(mbh)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte(
		`[["Content-Type","application/json"],["Content-Type","antani"],["Content-Length","17"]]`,
	)
	if !bytes.Equal(data, expected) {
		t.Fatal("result is unexpected")
	}
	var newlist HTTPHeadersList
	if err := json.Unmarshal(data, &newlist); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mbh, newlist) {
		t.Fatal("result mismatch")
	}
}

func TestMarshalUnmarshalHTTPHeaderBinary(t *testing.T) {
	mbh := HTTPHeadersList{
		HTTPHeader{
			Key: "Content-Type",
			Value: MaybeBinaryValue{
				Value: "application/json",
			},
		},
		HTTPHeader{
			Key: "Content-Type",
			Value: MaybeBinaryValue{
				Value: string(binaryInput),
			},
		},
		HTTPHeader{
			Key: "Content-Length",
			Value: MaybeBinaryValue{
				Value: "17",
			},
		},
	}
	data, err := json.Marshal(mbh)
	if err != nil {
		t.Fatal(err)
	}
	expected := []byte(
		`[["Content-Type","application/json"],["Content-Type",{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}],["Content-Length","17"]]`,
	)
	if !bytes.Equal(data, expected) {
		t.Fatal("result is unexpected")
	}
	var newlist HTTPHeadersList
	if err := json.Unmarshal(data, &newlist); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mbh, newlist) {
		t.Fatal("result mismatch")
	}
}

func TestHTTPHeaderUnmarshalJSON(t *testing.T) {
	t.Run("when the code is not a list", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`{"foo":1}`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the pair length is not two", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte("[1,2,3]")
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the first element is not a string", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`[1, "antani"]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the second element is not map[string]interface{}", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", ["base64", "foo"]]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the format field is missing", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the format field is not a string", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":1}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the format field is invalid", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":"antani"}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is missing", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":"base64"}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is not a string", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":"base64","data":10}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is not base64", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":"base64","data":"antani"}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
	t.Run("when the data field is not base64", func(t *testing.T) {
		var (
			hh    HTTPHeader
			input = []byte(`["antani", {"format":"base64","data":"antani"}]`)
		)
		if err := json.Unmarshal(input, &hh); err == nil {
			t.Fatal("expected an error here")
		}
	})
}

func TestNewDNSQueriesListEmpty(t *testing.T) {
	out := NewDNSQueriesList(oonitemplates.Results{})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewDNSQueriesListSuccess(t *testing.T) {
	out := NewDNSQueriesList(oonitemplates.Results{
		Resolves: []*modelx.ResolveDoneEvent{
			{
				Addresses: []string{
					"8.8.4.4", "2001:4860:4860::8888",
				},
				Hostname:         "dns.google",
				TransportNetwork: "system",
			},
			{
				Error:            errors.New(errorx.FailureDNSNXDOMAINError),
				Hostname:         "dns.googlex",
				TransportNetwork: "system",
			},
		},
	})
	if len(out) != 4 {
		t.Fatal("unexpected output length")
	}
	var (
		foundDNSGoogleA    bool
		foundDNSGoogleAAAA bool
		foundErrorA        bool
		foundErrorAAAA     bool
		foundOther         bool
	)
	for _, e := range out {
		switch e.Hostname {
		case "dns.google":
			switch e.QueryType {
			case "A":
				foundDNSGoogleA = true
				if err := dnscheckgood(e); err != nil {
					t.Fatal(err)
				}
			case "AAAA":
				foundDNSGoogleAAAA = true
				if err := dnscheckgood(e); err != nil {
					t.Fatal(err)
				}
			default:
				foundOther = true
			}
		case "dns.googlex":
			switch e.QueryType {
			case "A":
				foundErrorA = true
				if err := dnscheckbad(e); err != nil {
					t.Fatal(err)
				}
			case "AAAA":
				foundErrorAAAA = true
				if err := dnscheckbad(e); err != nil {
					t.Fatal(err)
				}
			default:
				foundOther = true
			}
		default:
			foundOther = true
		}
	}
	if foundDNSGoogleA == false {
		t.Fatal("missing A for dns.google")
	}
	if foundDNSGoogleAAAA == false {
		t.Fatal("missing AAAA for dns.google")
	}
	if foundErrorA == false {
		t.Fatal("missing A for invalid domain")
	}
	if foundErrorAAAA == false {
		t.Fatal("missing AAAA for invalid domain")
	}
	if foundOther == true {
		t.Fatal("seen something unexpected")
	}
}

func dnscheckgood(e DNSQueryEntry) error {
	if len(e.Answers) != 1 {
		return errors.New("unexpected number of answers")
	}
	if e.Engine != "system" {
		return errors.New("invalid engine")
	}
	if e.Failure != nil {
		return errors.New("invalid failure")
	}
	if e.Hostname != "dns.google" {
		return errors.New("invalid hostname")
	}
	switch e.QueryType {
	case "A", "AAAA":
	default:
		return errors.New("invalid query type")
	}
	if e.Answers[0].AnswerType != e.QueryType {
		return errors.New("AnswerType mismatch")
	}
	switch e.QueryType {
	case "A":
		if e.Answers[0].IPv4 != "8.8.4.4" {
			return errors.New("unexpected IPv4 entry")
		}
	case "AAAA":
		if e.Answers[0].IPv6 != "2001:4860:4860::8888" {
			return errors.New("unexpected IPv6 entry")
		}
	}
	if e.ResolverHostname != nil {
		return errors.New("invalid resolver hostname")
	}
	if e.ResolverPort != nil {
		return errors.New("invalid resolver port")
	}
	if e.ResolverAddress != "" {
		return errors.New("invalid resolver address")
	}
	return nil
}

func dnscheckbad(e DNSQueryEntry) error {
	if len(e.Answers) != 0 {
		return errors.New("unexpected number of answers")
	}
	if e.Engine != "system" {
		return errors.New("invalid engine")
	}
	if *e.Failure != errorx.FailureDNSNXDOMAINError {
		return errors.New("invalid failure")
	}
	if e.Hostname != "dns.googlex" {
		return errors.New("invalid hostname")
	}
	switch e.QueryType {
	case "A", "AAAA":
	default:
		return errors.New("invalid query type")
	}
	if e.ResolverHostname != nil {
		return errors.New("invalid resolver hostname")
	}
	if e.ResolverPort != nil {
		return errors.New("invalid resolver port")
	}
	if e.ResolverAddress != "" {
		return errors.New("invalid resolver address")
	}
	return nil
}

func TestDNSQueryTypeIPOfType(t *testing.T) {
	qtype := dnsQueryType("ANTANI")
	if qtype.ipoftype("8.8.8.8") == true {
		t.Fatal("ipoftype misbehaving")
	}
}

func TestNewNetworkEventsListEmpty(t *testing.T) {
	out := NewNetworkEventsList(oonitemplates.Results{})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewNetworkEventsListNoSuitableEvents(t *testing.T) {
	out := NewNetworkEventsList(oonitemplates.Results{
		NetworkEvents: []*modelx.Measurement{
			{},
			{},
			{},
		},
	})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewNetworkEventsListGood(t *testing.T) {
	out := NewNetworkEventsList(oonitemplates.Results{
		NetworkEvents: []*modelx.Measurement{
			{
				Connect: &modelx.ConnectEvent{
					DurationSinceBeginning: 10 * time.Millisecond,
					RemoteAddress:          "1.1.1.1:443",
				},
			},
			{
				Read: &modelx.ReadEvent{
					DurationSinceBeginning: 20 * time.Millisecond,
					NumBytes:               1789,
				},
			},
			{
				Write: &modelx.WriteEvent{
					DurationSinceBeginning: 30 * time.Millisecond,
					NumBytes:               17714,
				},
			},
		},
	})
	if len(out) != 3 {
		t.Fatal("unexpected output length")
	}

	if out[0].Address != "1.1.1.1:443" {
		t.Fatal("wrong out[0].Address")
	}
	if out[0].Failure != nil {
		t.Fatal("wrong out[0].Failure")
	}
	if out[0].NumBytes != 0 {
		t.Fatal("wrong out[0].NumBytes")
	}
	if out[0].Operation != errorx.ConnectOperation {
		t.Fatal("wrong out[0].Operation")
	}
	if !floatEquals(out[0].T, 0.010) {
		t.Fatal("wrong out[0].T")
	}

	if out[1].Address != "" {
		t.Fatal("wrong out[1].Address")
	}
	if out[1].Failure != nil {
		t.Fatal("wrong out[1].Failure")
	}
	if out[1].NumBytes != 1789 {
		t.Fatal("wrong out[1].NumBytes")
	}
	if out[1].Operation != errorx.ReadOperation {
		t.Fatal("wrong out[1].Operation")
	}
	if !floatEquals(out[1].T, 0.020) {
		t.Fatal("wrong out[1].T")
	}

	if out[2].Address != "" {
		t.Fatal("wrong out[2].Address")
	}
	if out[2].Failure != nil {
		t.Fatal("wrong out[2].Failure")
	}
	if out[2].NumBytes != 17714 {
		t.Fatal("wrong out[2].NumBytes")
	}
	if out[2].Operation != errorx.WriteOperation {
		t.Fatal("wrong out[2].Operation")
	}
	if !floatEquals(out[2].T, 0.030) {
		t.Fatal("wrong out[2].T")
	}
}

func TestNewNetworkEventsListGoodUDPAndErrors(t *testing.T) {
	out := NewNetworkEventsList(oonitemplates.Results{
		NetworkEvents: []*modelx.Measurement{
			{
				Connect: &modelx.ConnectEvent{
					DurationSinceBeginning: 10 * time.Millisecond,
					Error:                  errors.New("mocked error"),
					RemoteAddress:          "1.1.1.1:443",
				},
			},
			{
				Read: &modelx.ReadEvent{
					DurationSinceBeginning: 20 * time.Millisecond,
					Error:                  errors.New("mocked error"),
					NumBytes:               1789,
				},
			},
			{
				Write: &modelx.WriteEvent{
					DurationSinceBeginning: 30 * time.Millisecond,
					Error:                  errors.New("mocked error"),
					NumBytes:               17714,
				},
			},
		},
	})
	if len(out) != 3 {
		t.Fatal("unexpected output length")
	}

	if out[0].Address != "1.1.1.1:443" {
		t.Fatal("wrong out[0].Address")
	}
	if *out[0].Failure != "mocked error" {
		t.Fatal("wrong out[0].Failure")
	}
	if out[0].NumBytes != 0 {
		t.Fatal("wrong out[0].NumBytes")
	}
	if out[0].Operation != errorx.ConnectOperation {
		t.Fatal("wrong out[0].Operation")
	}
	if !floatEquals(out[0].T, 0.010) {
		t.Fatal("wrong out[0].T")
	}

	if out[1].Address != "" {
		t.Fatal("wrong out[1].Address")
	}
	if *out[1].Failure != "mocked error" {
		t.Fatal("wrong out[1].Failure")
	}
	if out[1].NumBytes != 1789 {
		t.Fatal("wrong out[1].NumBytes")
	}
	if out[1].Operation != errorx.ReadOperation {
		t.Fatal("wrong out[1].Operation")
	}
	if !floatEquals(out[1].T, 0.020) {
		t.Fatal("wrong out[1].T")
	}

	if out[2].Address != "" {
		t.Fatal("wrong out[2].Address")
	}
	if *out[2].Failure != "mocked error" {
		t.Fatal("wrong out[2].Failure")
	}
	if out[2].NumBytes != 17714 {
		t.Fatal("wrong out[2].NumBytes")
	}
	if out[2].Operation != errorx.WriteOperation {
		t.Fatal("wrong out[2].Operation")
	}
	if !floatEquals(out[2].T, 0.030) {
		t.Fatal("wrong out[2].T")
	}
}

func floatEquals(a, b float64) bool {
	const c = 1e-03
	return (a-b) < c && (b-a) < c
}

func TestNewTLSHandshakesListEmpty(t *testing.T) {
	out := NewTLSHandshakesList(oonitemplates.Results{})
	if len(out) != 0 {
		t.Fatal("unexpected output length")
	}
}

func TestNewTLSHandshakesListSuccess(t *testing.T) {
	out := NewTLSHandshakesList(oonitemplates.Results{
		TLSHandshakes: []*modelx.TLSHandshakeDoneEvent{
			{},
			{
				Error: errors.New("mocked error"),
			},
			{
				ConnectionState: modelx.TLSConnectionState{
					CipherSuite:        tls.TLS_AES_128_GCM_SHA256,
					NegotiatedProtocol: "h2",
					PeerCertificates: []modelx.X509Certificate{
						{
							Data: []byte("deadbeef"),
						},
						{
							Data: []byte("abad1dea"),
						},
					},
					Version: tls.VersionTLS11,
				},
				DurationSinceBeginning: 10 * time.Millisecond,
			},
		},
	})
	if len(out) != 3 {
		t.Fatal("unexpected output length")
	}

	if out[0].CipherSuite != "" {
		t.Fatal("invalid out[0].CipherSuite")
	}
	if out[0].Failure != nil {
		t.Fatal("invalid out[0].Failure")
	}
	if out[0].NegotiatedProtocol != "" {
		t.Fatal("invalid out[0].NegotiatedProtocol")
	}
	if len(out[0].PeerCertificates) != 0 {
		t.Fatal("invalid out[0].PeerCertificates")
	}
	if !floatEquals(out[0].T, 0) {
		t.Fatal("invalid out[0].T")
	}
	if out[0].TLSVersion != "" {
		t.Fatal("invalid out[0].TLSVersion")
	}

	if out[1].CipherSuite != "" {
		t.Fatal("invalid out[1].CipherSuite")
	}
	if *out[1].Failure != "mocked error" {
		t.Fatal("invalid out[1].Failure")
	}
	if out[1].NegotiatedProtocol != "" {
		t.Fatal("invalid out[1].NegotiatedProtocol")
	}
	if len(out[1].PeerCertificates) != 0 {
		t.Fatal("invalid out[1].PeerCertificates")
	}
	if !floatEquals(out[1].T, 0) {
		t.Fatal("invalid out[1].T")
	}
	if out[1].TLSVersion != "" {
		t.Fatal("invalid out[1].TLSVersion")
	}

	if out[2].CipherSuite != "TLS_AES_128_GCM_SHA256" {
		t.Fatal("invalid out[2].CipherSuite")
	}
	if out[2].Failure != nil {
		t.Fatal("invalid out[2].Failure")
	}
	if out[2].NegotiatedProtocol != "h2" {
		t.Fatal("invalid out[2].NegotiatedProtocol")
	}
	if len(out[2].PeerCertificates) != 2 {
		t.Fatal("invalid out[2].PeerCertificates")
	}
	if !floatEquals(out[2].T, 0.010) {
		t.Fatal("invalid out[2].T")
	}
	if out[2].TLSVersion != "TLSv1.1" {
		t.Fatal("invalid out[2].TLSVersion")
	}

	for idx, mbv := range out[2].PeerCertificates {
		if idx == 0 && mbv.Value != "deadbeef" {
			t.Fatal("invalid first certificate")
		}
		if idx == 1 && mbv.Value != "abad1dea" {
			t.Fatal("invalid second certificate")
		}
		if idx < 0 || idx > 1 {
			t.Fatal("invalid index")
		}
	}
}
