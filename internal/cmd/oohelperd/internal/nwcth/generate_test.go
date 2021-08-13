package nwcth

import (
	"context"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var generator = &DefaultGenerator{resolver: newResolver()}

func TestGenerateDNSFailure(t *testing.T) {
	u, err := url.Parse("https://www.google.google")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			proto: "https",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			sortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if len(urlMeasurements) != 1 {
		t.Fatal("unexpected urlMeasurements length")
	}

}

func TestGenerate(t *testing.T) {
	u, err := url.Parse("http://www.google.com")
	u2, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			proto: "http",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			sortIndex: 0,
		},
		{
			proto: "https",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			sortIndex: 0,
		},
		{
			proto: "h3",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			sortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts)
	if err != nil {
		t.Fatal("unexpected err")
	}
	if urlMeasurements == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if len(urlMeasurements) < 3 {
		t.Fatal("unexpected number of urlMeasurements", len(urlMeasurements))
	}
}

func TestGenerateUnexpectedProtocol(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			proto: "h3-27",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			sortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts)
	if err != nil {
		t.Fatal("unexpected err")
	}
	if urlMeasurements == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if len(urlMeasurements) != 1 {
		t.Fatal("unexpected number of urlMeasurements")
	}
	measurement := urlMeasurements[0]
	if measurement.URL != u.String() {
		t.Fatal("unexpected URL")
	}
	if measurement.DNS == nil {
		t.Fatal("DNS should not be nil")
	}
	if measurement.RoundTrip == nil {
		t.Fatal("RoundTrip should not be nil")
	}
	if measurement.Endpoints != nil {
		t.Fatal("Endpoints should be nil")
	}
}

func TestGenerateHTTP(t *testing.T) {
	u, err := url.Parse("http://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		proto: "http",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		sortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPEndpoint(context.Background(), rt, "93.184.216.34:80")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if reflect.TypeOf(endpointMeasurement).String() != "*nwcth.HTTPEndpointMeasurement" {
		t.Fatal("unexpected type")
	}
	if endpointMeasurement.(*HTTPEndpointMeasurement).TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*HTTPEndpointMeasurement).HTTPRoundtripMeasurement == nil {
		t.Fatal("HTTPRoundtripMeasurement should not be nil")
	}
}

func TestGenerateHTTPS(t *testing.T) {
	u, err := url.Parse("https://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		proto: "https",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		sortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPSEndpoint(context.Background(), rt, "93.184.216.34:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if reflect.TypeOf(endpointMeasurement).String() != "*nwcth.HTTPSEndpointMeasurement" {
		t.Fatal("unexpected type")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).TLSHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).HTTPRoundtripMeasurement == nil {
		t.Fatal("HTTPRoundtripMeasurement should not be nil")
	}
}

func TestGenerateHTTPSTLSFailure(t *testing.T) {
	u, err := url.Parse("https://wrong.host.badssl.com/")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		proto: "https",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		sortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPSEndpoint(context.Background(), rt, "104.154.89.105:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if reflect.TypeOf(endpointMeasurement).String() != "*nwcth.HTTPSEndpointMeasurement" {
		t.Fatal("unexpected type")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).TLSHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*HTTPSEndpointMeasurement).HTTPRoundtripMeasurement != nil {
		t.Fatal("HTTPRoundtripMeasurement should be nil")
	}
}

func TestGenerateH3(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		proto: "h3",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		sortIndex: 0,
	}
	endpointMeasurement := generator.GenerateH3Endpoint(context.Background(), rt, "173.194.76.103:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if reflect.TypeOf(endpointMeasurement).String() != "*nwcth.H3EndpointMeasurement" {
		t.Fatal("unexpected type")
	}
	if endpointMeasurement.(*H3EndpointMeasurement).QUICHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.(*H3EndpointMeasurement).HTTPRoundtripMeasurement == nil {
		t.Fatal("HTTPRoundtripMeasurement should not be nil")
	}
}
