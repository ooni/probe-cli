package netxlite

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestHTTPTransportLogger(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("with failure", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebug: func(message string) {
					count++
				},
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			txp := &httpTransportLogger{
				Logger: lo,
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
			}
			client := &http.Client{Transport: txp}
			resp, err := client.Get("https://www.google.com")
			if !errors.Is(err, io.EOF) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil response here")
			}
			if count < 1 {
				t.Fatal("no logs?!")
			}
		})

		t.Run("with success", func(t *testing.T) {
			var count int
			lo := &mocks.Logger{
				MockDebug: func(message string) {
					count++
				},
				MockDebugf: func(format string, v ...interface{}) {
					count++
				},
			}
			txp := &httpTransportLogger{
				Logger: lo,
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return &http.Response{
							Body: io.NopCloser(strings.NewReader("")),
							Header: http.Header{
								"Server": []string{"antani/0.1.0"},
							},
							StatusCode: 200,
						}, nil
					},
				},
			}
			client := &http.Client{Transport: txp}
			req, err := http.NewRequest("GET", "https://www.google.com", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("User-Agent", "miniooni/0.1.0-dev")
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			ReadAllContext(context.Background(), LimitBodyReader(resp))
			resp.Body.Close()
			if count < 1 {
				t.Fatal("no logs?!")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		calls := &atomic.Int64{}
		txp := &httpTransportLogger{
			HTTPTransport: &mocks.HTTPTransport{
				MockCloseIdleConnections: func() {
					calls.Add(1)
				},
			},
			Logger: log.Log,
		}
		txp.CloseIdleConnections()
		if calls.Load() != 1 {
			t.Fatal("not called")
		}
	})
}
