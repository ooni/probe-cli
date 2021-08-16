package nwcth

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var generator = &DefaultGenerator{resolver: newResolver()}

type fakeTransport struct {
	err error
}

func (txp fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, txp.err
}
func (txp fakeTransport) CloseIdleConnections() {}

type fakeQUICDialer struct {
	err error
}

func (d fakeQUICDialer) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	return nil, d.err
}

type fakeDialer struct {
	err error
}

func (d fakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return nil, d.err
}

func TestGenerateDNSFailure(t *testing.T) {
	u, err := url.Parse("https://www.google.google")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			Proto: "https",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts, []string{})
	if err != nil {
		t.Fatal("unexpected error")
	}
	if len(urlMeasurements) != 1 {
		t.Fatal("unexpected urlMeasurements length")
	}
	if urlMeasurements[0].DNS == nil {
		t.Fatal("DNS should not be nil")
	}
	if urlMeasurements[0].DNS.Failure == nil || *urlMeasurements[0].DNS.Failure != errorsx.FailureDNSNXDOMAINError {
		t.Fatal("unexpected DNS failure type")
	}
}

func TestGenerate(t *testing.T) {
	u, err := url.Parse("http://www.google.com")
	u2, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			Proto: "http",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
		{
			Proto: "https",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
		{
			Proto: "h3",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts, []string{})
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
			Proto: "h3-27",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts, []string{})
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

func TestGenerateURLWithClientResolutions(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "h3",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	clientResolution := "142.250.186.36"
	urlMeasurement := generator.GenerateURL(context.Background(), rt, []string{clientResolution})
	if err != nil {
		t.Fatal("unexpected err")
	}
	if urlMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if urlMeasurement.DNS == nil {
		t.Fatal("DNS should not be nil")
	}
	if len(urlMeasurement.Endpoints) < 2 {
		t.Fatal("unexpected number of endpoints")
	}
	clientAddrsFound := false
	for _, e := range urlMeasurement.Endpoints {
		if e.Endpoint == clientResolution+":443" {
			clientAddrsFound = true
		}
	}
	if !clientAddrsFound {
		t.Fatal("did not use provided client resolution")
	}
}

func TestGenerateHTTP(t *testing.T) {
	u, err := url.Parse("http://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "http",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPEndpoint(context.Background(), rt, "93.184.216.34:80")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if endpointMeasurement.TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.HTTPRoundTripMeasurement == nil {
		t.Fatal("HTTPRoundTripMeasurement should not be nil")
	}
}

func TestGenerateHTTPS(t *testing.T) {
	u, err := url.Parse("https://example.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "https",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPSEndpoint(context.Background(), rt, "93.184.216.34:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if endpointMeasurement.TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.TLSHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.HTTPRoundTripMeasurement == nil {
		t.Fatal("HTTPRoundTripMeasurement should not be nil")
	}
}

func TestGenerateHTTPSTLSFailure(t *testing.T) {
	u, err := url.Parse("https://wrong.host.badssl.com/")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "https",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPSEndpoint(context.Background(), rt, "104.154.89.105:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if endpointMeasurement.TCPConnectMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.TLSHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.HTTPRoundTripMeasurement != nil {
		t.Fatal("HTTPRoundTripMeasurement should be nil")
	}
}

func TestGenerateH3(t *testing.T) {
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "h3",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateH3Endpoint(context.Background(), rt, "173.194.76.103:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement == nil {
		t.Fatal("unexpected nil urlMeasurement")
	}
	if endpointMeasurement.QUICHandshakeMeasurement == nil {
		t.Fatal("TCPConnectMeasurement should not be nil")
	}
	if endpointMeasurement.HTTPRoundTripMeasurement == nil {
		t.Fatal("HTTPRoundTripMeasurement should not be nil")
	}
}

func TestGenerateTCPDoFails(t *testing.T) {
	expected := errors.New("expected")
	generator := &DefaultGenerator{
		dialer:   fakeDialer{err: expected},
		resolver: newResolver(),
	}
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "https",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateHTTPSEndpoint(context.Background(), rt, "173.194.76.103:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement.TCPConnectMeasurement == nil {
		t.Fatal("QUIC handshake should not be nil")
	}
	if endpointMeasurement.TCPConnectMeasurement.Failure == nil {
		t.Fatal("expected an error here")
	}
	if *endpointMeasurement.TCPConnectMeasurement.Failure != *newfailure(expected) {
		t.Fatal("unexpected error type")
	}
}

func TestGenerateQUICDoFails(t *testing.T) {
	expected := errors.New("expected")
	generator := &DefaultGenerator{
		quicDialer: fakeQUICDialer{err: expected},
		resolver:   newResolver(),
	}
	u, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rt := &RoundTrip{
		Proto: "h3",
		Request: &http.Request{
			URL: u,
		},
		Response: &http.Response{
			StatusCode: 200,
		},
		SortIndex: 0,
	}
	endpointMeasurement := generator.GenerateH3Endpoint(context.Background(), rt, "173.194.76.103:443")
	if err != nil {
		t.Fatal("unexpected err")
	}
	if endpointMeasurement.QUICHandshakeMeasurement == nil {
		t.Fatal("QUIC handshake should not be nil")
	}
	if endpointMeasurement.QUICHandshakeMeasurement.Failure == nil {
		t.Fatal("expected an error here")
	}
	if *endpointMeasurement.QUICHandshakeMeasurement.Failure != *newfailure(expected) {
		t.Fatal("unexpected error type")
	}
}

func TestGenerateHTTPDoFails(t *testing.T) {
	expected := errors.New("expected")
	generator := &DefaultGenerator{
		transport: fakeTransport{err: expected},
		resolver:  newResolver(),
	}
	u, err := url.Parse("http://www.google.com")
	u2, err := url.Parse("https://www.google.com")
	runtimex.PanicOnError(err, "url.Parse failed")
	rts := []*RoundTrip{
		{
			Proto: "http",
			Request: &http.Request{
				URL: u,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
		{
			Proto: "https",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
		{
			Proto: "h3",
			Request: &http.Request{
				URL: u2,
			},
			Response: &http.Response{
				StatusCode: 200,
			},
			SortIndex: 0,
		},
	}
	urlMeasurements, err := generator.Generate(context.Background(), rts, []string{})
	if err != nil {
		t.Fatal("unexpected err")
	}
	if len(urlMeasurements) != 3 {
		t.Fatal("unexpected number of urlMeasurements")
	}
	for _, u := range urlMeasurements {
		if u.DNS == nil {
			t.Fatal("unexpected DNS failure")
		}
		if len(u.Endpoints) < 1 {
			t.Fatal("unexpected number of endpoints", len(u.Endpoints))
		}
		if u.Endpoints[0].HTTPRoundTripMeasurement == nil {
			t.Fatal("roundtrip should not be nil")
		}
		if u.Endpoints[0].HTTPRoundTripMeasurement.Response == nil {
			t.Fatal("roundtrip response should not be nil")
		}
		if u.Endpoints[0].HTTPRoundTripMeasurement.Response.Failure == nil {
			t.Fatal("expected an HTTP error")
		}
		if !strings.HasSuffix(*u.Endpoints[0].HTTPRoundTripMeasurement.Response.Failure, expected.Error()) {
			t.Fatal("unexpected failure type")
		}
	}
}
