package httptransport_test

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverMetadataSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	txp := httptransport.SaverMetadataHTTPTransport{
		HTTPTransport: netxlite.NewHTTPTransportStdlib(model.DiscardLogger),
		Saver:         saver,
	}
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("User-Agent", "miniooni/0.1.0-dev")
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal("not the error we expected")
	}
	if resp == nil {
		t.Fatal("expected non nil response here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected two events")
	}
	//
	if ev[0].HTTPMethod != "GET" {
		t.Fatal("unexpected Method")
	}
	if len(ev[0].HTTPHeaders) <= 0 {
		t.Fatal("unexpected Headers")
	}
	if ev[0].HTTPURL != "https://www.google.com" {
		t.Fatal("unexpected URL")
	}
	if ev[0].Name != "http_request_metadata" {
		t.Fatal("unexpected Name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("unexpected Time")
	}
	//
	if ev[1].HTTPStatusCode != 200 {
		t.Fatal("unexpected StatusCode")
	}
	if len(ev[1].HTTPHeaders) <= 0 {
		t.Fatal("unexpected Headers")
	}
	if ev[1].Name != "http_response_metadata" {
		t.Fatal("unexpected Name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverMetadataFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := &trace.Saver{}
	txp := httptransport.SaverMetadataHTTPTransport{
		HTTPTransport: FakeTransport{
			Err: expected,
		},
		Saver: saver,
	}
	req, err := http.NewRequest("GET", "http://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Add("User-Agent", "miniooni/0.1.0-dev")
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	ev := saver.Read()
	if len(ev) != 1 {
		t.Fatal("expected one event")
	}
	if ev[0].HTTPMethod != "GET" {
		t.Fatal("unexpected Method")
	}
	if len(ev[0].HTTPHeaders) <= 0 {
		t.Fatal("unexpected Headers")
	}
	if ev[0].HTTPURL != "http://www.google.com" {
		t.Fatal("unexpected URL")
	}
	if ev[0].Name != "http_request_metadata" {
		t.Fatal("unexpected Name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverTransactionSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	txp := httptransport.SaverTransactionHTTPTransport{
		HTTPTransport: netxlite.NewHTTPTransportStdlib(model.DiscardLogger),
		Saver:         saver,
	}
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal("not the error we expected")
	}
	if resp == nil {
		t.Fatal("expected non nil response here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected two events")
	}
	//
	if ev[0].Name != "http_transaction_start" {
		t.Fatal("unexpected Name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("unexpected Time")
	}
	//
	if ev[1].Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[1].Name != "http_transaction_done" {
		t.Fatal("unexpected Name")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverTransactionFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := &trace.Saver{}
	txp := httptransport.SaverTransactionHTTPTransport{
		HTTPTransport: FakeTransport{
			Err: expected,
		},
		Saver: saver,
	}
	req, err := http.NewRequest("GET", "http://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("expected two events")
	}
	if ev[0].Name != "http_transaction_start" {
		t.Fatal("unexpected Name")
	}
	if !ev[0].Time.Before(time.Now()) {
		t.Fatal("unexpected Time")
	}
	if ev[1].Name != "http_transaction_done" {
		t.Fatal("unexpected Name")
	}
	if !errors.Is(ev[1].Err, expected) {
		t.Fatal("unexpected Err")
	}
	if !ev[1].Time.After(ev[0].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverBodySuccess(t *testing.T) {
	saver := new(trace.Saver)
	txp := httptransport.SaverBodyHTTPTransport{
		HTTPTransport: FakeTransport{
			Func: func(req *http.Request) (*http.Response, error) {
				data, err := netxlite.ReadAllContext(context.Background(), req.Body)
				if err != nil {
					t.Fatal(err)
				}
				if string(data) != "deadbeef" {
					t.Fatal("invalid data")
				}
				return &http.Response{
					StatusCode: 501,
					Body:       io.NopCloser(strings.NewReader("abad1dea")),
				}, nil
			},
		},
		SnapshotSize: 4,
		Saver:        saver,
	}
	body := strings.NewReader("deadbeef")
	req, err := http.NewRequest("POST", "http://x.org/y", body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 501 {
		t.Fatal("unexpected status code")
	}
	defer resp.Body.Close()
	data, err := netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "abad1dea" {
		t.Fatal("unexpected body")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("unexpected number of events")
	}
	if string(ev[0].Data) != "dead" {
		t.Fatal("invalid Data")
	}
	if ev[0].DataIsTruncated != true {
		t.Fatal("invalid DataIsTruncated")
	}
	if ev[0].Name != "http_request_body_snapshot" {
		t.Fatal("invalid Name")
	}
	if ev[0].Time.After(time.Now()) {
		t.Fatal("invalid Time")
	}
	if string(ev[1].Data) != "abad" {
		t.Fatal("invalid Data")
	}
	if ev[1].DataIsTruncated != true {
		t.Fatal("invalid DataIsTruncated")
	}
	if ev[1].Name != "http_response_body_snapshot" {
		t.Fatal("invalid Name")
	}
	if ev[1].Time.Before(ev[0].Time) {
		t.Fatal("invalid Time")
	}
}

func TestSaverBodyRequestReadError(t *testing.T) {
	saver := new(trace.Saver)
	txp := httptransport.SaverBodyHTTPTransport{
		HTTPTransport: FakeTransport{
			Func: func(req *http.Request) (*http.Response, error) {
				panic("should not be called")
			},
		},
		SnapshotSize: 4,
		Saver:        saver,
	}
	expected := errors.New("mocked error")
	body := FakeBody{Err: expected}
	req, err := http.NewRequest("POST", "http://x.org/y", body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	ev := saver.Read()
	if len(ev) != 0 {
		t.Fatal("unexpected number of events")
	}
}

func TestSaverBodyRoundTripError(t *testing.T) {
	saver := new(trace.Saver)
	expected := errors.New("mocked error")
	txp := httptransport.SaverBodyHTTPTransport{
		HTTPTransport: FakeTransport{
			Err: expected,
		},
		SnapshotSize: 4,
		Saver:        saver,
	}
	body := strings.NewReader("deadbeef")
	req, err := http.NewRequest("POST", "http://x.org/y", body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	ev := saver.Read()
	if len(ev) != 1 {
		t.Fatal("unexpected number of events")
	}
	if string(ev[0].Data) != "dead" {
		t.Fatal("invalid Data")
	}
	if ev[0].DataIsTruncated != true {
		t.Fatal("invalid DataIsTruncated")
	}
	if ev[0].Name != "http_request_body_snapshot" {
		t.Fatal("invalid Name")
	}
	if ev[0].Time.After(time.Now()) {
		t.Fatal("invalid Time")
	}
}

func TestSaverBodyResponseReadError(t *testing.T) {
	saver := new(trace.Saver)
	expected := errors.New("mocked error")
	txp := httptransport.SaverBodyHTTPTransport{
		HTTPTransport: FakeTransport{
			Func: func(req *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: 200,
					Body: FakeBody{
						Err: expected,
					},
				}, nil
			},
		},
		SnapshotSize: 4,
		Saver:        saver,
	}
	body := strings.NewReader("deadbeef")
	req, err := http.NewRequest("POST", "http://x.org/y", body)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	ev := saver.Read()
	if len(ev) != 1 {
		t.Fatal("unexpected number of events")
	}
	if string(ev[0].Data) != "dead" {
		t.Fatal("invalid Data")
	}
	if ev[0].DataIsTruncated != true {
		t.Fatal("invalid DataIsTruncated")
	}
	if ev[0].Name != "http_request_body_snapshot" {
		t.Fatal("invalid Name")
	}
	if ev[0].Time.After(time.Now()) {
		t.Fatal("invalid Time")
	}
}

func TestCloneHeaders(t *testing.T) {
	t.Run("with req.Host set", func(t *testing.T) {
		req := &http.Request{
			Host: "www.example.com",
			URL: &url.URL{
				Host: "www.kernel.org",
			},
			Header: http.Header{},
		}
		txp := httptransport.SaverMetadataHTTPTransport{}
		header := txp.CloneHeaders(req)
		if header.Get("Host") != "www.example.com" {
			t.Fatal("did not set Host header correctly")
		}
	})

	t.Run("with only req.URL.Host set", func(t *testing.T) {
		req := &http.Request{
			Host: "",
			URL: &url.URL{
				Host: "www.kernel.org",
			},
			Header: http.Header{},
		}
		txp := httptransport.SaverMetadataHTTPTransport{}
		header := txp.CloneHeaders(req)
		if header.Get("Host") != "www.kernel.org" {
			t.Fatal("did not set Host header correctly")
		}
	})
}

type FakeDialer struct {
	Conn net.Conn
	Err  error
}

func (d FakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	time.Sleep(10 * time.Microsecond)
	return d.Conn, d.Err
}

type FakeTransport struct {
	Name string
	Err  error
	Func func(*http.Request) (*http.Response, error)
	Resp *http.Response
}

func (txp FakeTransport) Network() string {
	return txp.Name
}

func (txp FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	if txp.Func != nil {
		return txp.Func(req)
	}
	if req.Body != nil {
		netxlite.ReadAllContext(req.Context(), req.Body)
		req.Body.Close()
	}
	if txp.Err != nil {
		return nil, txp.Err
	}
	txp.Resp.Request = req // non thread safe but it doesn't matter
	return txp.Resp, nil
}

func (txp FakeTransport) CloseIdleConnections() {}

type FakeBody struct {
	Err error
}

func (fb FakeBody) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	return 0, fb.Err
}

func (fb FakeBody) Close() error {
	return nil
}
