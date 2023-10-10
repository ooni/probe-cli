package testingproxy

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"
)

func TestHTTPClientMock(t *testing.T) {
	t.Run("for Get", func(t *testing.T) {
		expected := errors.New("mocked error")
		c := &httpClientMock{
			MockGet: func(URL string) (*http.Response, error) {
				return nil, expected
			},
		}
		resp, err := c.Get("https://www.google.com/")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error")
		}
		if resp != nil {
			t.Fatal("expected nil response")
		}
	})
}

func TestHTTPTestingTMock(t *testing.T) {
	t.Run("for Fatal", func(t *testing.T) {
		var called bool
		mt := &httpTestingTMock{
			MockFatal: func(v ...any) {
				called = true
			},
		}
		mt.Fatal("antani")
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("for Logf", func(t *testing.T) {
		var called bool
		mt := &httpTestingTMock{
			MockLogf: func(format string, v ...any) {
				called = true
			},
		}
		mt.Logf("antani %v", "mascetti")
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestHTTPCheckResponseHandlesFailures(t *testing.T) {
	type testcase struct {
		name      string
		mclient   httpClient
		expectLog bool
	}

	testcases := []testcase{{
		name: "when HTTP round trip fails",
		mclient: &httpClientMock{
			MockGet: func(URL string) (*http.Response, error) {
				return nil, io.EOF
			},
		},
		expectLog: false,
	}, {
		name: "with unexpected status code",
		mclient: &httpClientMock{
			MockGet: func(URL string) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 404,
					Body:       io.NopCloser(bytes.NewReader(nil)),
				}
				return resp, nil
			},
		},
		expectLog: true,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {

			// prepare for capturing what happened
			var (
				calledLogf  bool
				calledFatal bool
			)
			mt := &httpTestingTMock{
				MockLogf: func(format string, v ...any) {
					calledLogf = true
				},
				MockFatal: func(v ...any) {
					calledFatal = true
					panic(v)
				},
			}

			// make sure we handle the panic and check what happened
			defer func() {
				result := recover()
				if result == nil {
					t.Fatal("did not panic")
				}
				if !calledFatal {
					t.Fatal("did not actually call t.Fatal")
				}
				if tc.expectLog != calledLogf {
					t.Fatal("tc.expectLog is", tc.expectLog, "but calledLogf is", calledLogf)
				}
			}()

			// invoke the function we're testing
			httpCheckResponse(mt, tc.mclient, "https://www.google.com/")
		})
	}
}
