package netxlite

import (
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestHTTPTransportErrWrapper(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("with failure", func(t *testing.T) {
			txp := &httpTransportErrWrapper{
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
			}
			resp, err := txp.RoundTrip(&http.Request{})
			var errWrapper *ErrWrapper
			if !errors.As(err, &errWrapper) {
				t.Fatal("the returned error is not an ErrWrapper")
			}
			if errWrapper.Failure != FailureEOFError {
				t.Fatal("unexpected failure", errWrapper.Failure)
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
		})

		t.Run("with success", func(t *testing.T) {
			expect := &http.Response{}
			txp := &httpTransportErrWrapper{
				HTTPTransport: &mocks.HTTPTransport{
					MockRoundTrip: func(req *http.Request) (*http.Response, error) {
						return expect, nil
					},
				},
			}
			resp, err := txp.RoundTrip(&http.Request{})
			if err != nil {
				t.Fatal(err)
			}
			if resp != expect {
				t.Fatal("not the expected response")
			}
		})
	})
}

func TestHTTPClientErrWrapper(t *testing.T) {
	t.Run("Do", func(t *testing.T) {
		t.Run("with failure", func(t *testing.T) {
			clnt := &httpClientErrWrapper{
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						return nil, io.EOF
					},
				},
			}
			resp, err := clnt.Do(&http.Request{})
			var errWrapper *ErrWrapper
			if !errors.As(err, &errWrapper) {
				t.Fatal("the returned error is not an ErrWrapper")
			}
			if errWrapper.Failure != FailureEOFError {
				t.Fatal("unexpected failure", errWrapper.Failure)
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
		})

		t.Run("with success", func(t *testing.T) {
			expect := &http.Response{}
			clnt := &httpClientErrWrapper{
				HTTPClient: &mocks.HTTPClient{
					MockDo: func(req *http.Request) (*http.Response, error) {
						return expect, nil
					},
				},
			}
			resp, err := clnt.Do(&http.Request{})
			if err != nil {
				t.Fatal(err)
			}
			if resp != expect {
				t.Fatal("not the expected response")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.HTTPClient{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		clnt := &httpClientErrWrapper{child}
		clnt.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}
