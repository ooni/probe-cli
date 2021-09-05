package httptransport_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
)

func TestByteCounterFailure(t *testing.T) {
	counter := bytecounter.New()
	txp := httptransport.ByteCountingTransport{
		Counter: counter,
		RoundTripper: httptransport.FakeTransport{
			Err: io.EOF,
		},
	}
	client := &http.Client{Transport: txp}
	req, err := http.NewRequest(
		"POST", "https://www.google.com", strings.NewReader("AAAAAA"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "antani-browser/1.0.0")
	resp, err := client.Do(req)
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
	if counter.Sent.Load() != 68 {
		t.Fatal("expected around 68 bytes sent")
	}
	if counter.Received.Load() != 0 {
		t.Fatal("expected zero bytes received")
	}
}

func TestByteCounterSuccess(t *testing.T) {
	counter := bytecounter.New()
	txp := httptransport.ByteCountingTransport{
		Counter: counter,
		RoundTripper: httptransport.FakeTransport{
			Resp: &http.Response{
				Body: io.NopCloser(strings.NewReader("1234567")),
				Header: http.Header{
					"Server": []string{"antani/0.1.0"},
				},
				Status:     "200 OK",
				StatusCode: http.StatusOK,
			},
		},
	}
	client := &http.Client{Transport: txp}
	req, err := http.NewRequest(
		"POST", "https://www.google.com", strings.NewReader("AAAAAA"))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("User-Agent", "antani-browser/1.0.0")
	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	data, err := iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if string(data) != "1234567" {
		t.Fatal("expected a different body here")
	}
	if counter.Sent.Load() != 68 {
		t.Fatal("expected around 68 bytes sent")
	}
	if counter.Received.Load() != 37 {
		t.Fatal("expected zero around 37 bytes received")
	}
}

func TestByteCounterSuccessWithEOF(t *testing.T) {
	counter := bytecounter.New()
	txp := httptransport.ByteCountingTransport{
		Counter: counter,
		RoundTripper: httptransport.FakeTransport{
			Resp: &http.Response{
				Body: bodyReaderWithEOF{},
				Header: http.Header{
					"Server": []string{"antani/0.1.0"},
				},
				Status:     "200 OK",
				StatusCode: http.StatusOK,
			},
		},
	}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	data, err := iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if string(data) != "A" {
		t.Fatal("expected a different body here")
	}
}

type bodyReaderWithEOF struct{}

func (bodyReaderWithEOF) Read(p []byte) (int, error) {
	if len(p) < 1 {
		panic("should not happen")
	}
	p[0] = 'A'
	return 1, io.EOF // we want code to be robust to this
}
func (bodyReaderWithEOF) Close() error {
	return nil
}
