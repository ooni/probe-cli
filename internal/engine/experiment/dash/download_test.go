package dash

import (
	"bytes"
	"context"
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestDownloadNewHTTPRequestFailure(t *testing.T) {
	expected := errors.New("mocked error")
	_, err := download(context.Background(), downloadConfig{
		deps: FakeDeps{newHTTPRequestErr: expected},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadHTTPClientDoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	txp := FakeHTTPTransport{err: expected}
	_, err := download(context.Background(), downloadConfig{
		deps: FakeDeps{httpTransport: txp, newHTTPRequestResult: &http.Request{
			Header: http.Header{},
			URL:    &url.URL{},
		}},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadInternalError(t *testing.T) {
	txp := FakeHTTPTransport{resp: &http.Response{StatusCode: 500}}
	_, err := download(context.Background(), downloadConfig{
		deps: FakeDeps{httpTransport: txp, newHTTPRequestResult: &http.Request{
			Header: http.Header{},
			URL:    &url.URL{},
		}},
	})
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadReadAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	txp := FakeHTTPTransport{resp: &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
		StatusCode: 200,
	}}
	_, err := download(context.Background(), downloadConfig{
		deps: FakeDeps{
			httpTransport: txp,
			newHTTPRequestResult: &http.Request{
				Header: http.Header{},
				URL:    &url.URL{},
			},
			readAllErr: expected,
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadSuccess(t *testing.T) {
	txp := FakeHTTPTransport{resp: &http.Response{
		Body:       ioutil.NopCloser(bytes.NewReader(nil)),
		StatusCode: 200,
	}}
	result, err := download(context.Background(), downloadConfig{
		deps: FakeDeps{
			httpTransport: txp,
			newHTTPRequestResult: &http.Request{
				Header: http.Header{},
				URL:    &url.URL{},
			},
			readAllResult: []byte("[]"),
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.elapsed <= 0 {
		t.Fatal("invalid elapsed")
	}
	if result.received <= 0 {
		t.Fatal("invalid received")
	}
	if result.requestTicks <= 0 {
		t.Fatal("invalid requestTicks")
	}
	if result.serverURL == "" {
		t.Fatal("invalid serverURL")
	}
	if result.timestamp <= 0 {
		t.Fatal("invalid timestamp")
	}
}
