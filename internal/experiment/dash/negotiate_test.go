package dash

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNegotiateNewHTTPRequestWithContextFailure(t *testing.T) {
	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
	}

	result, err := negotiate(context.Background(), "\t", deps)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateHTTPClientDoFailure(t *testing.T) {
	expected := errors.New("mocked error")

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					return nil, expected
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateInternalError(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 500,
						Body:       io.NopCloser(strings.NewReader("")),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateReadAllFailure(t *testing.T) {
	expected := errors.New("mocked error")

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					reader := &mocks.Reader{
						MockRead: func(b []byte) (int, error) {
							return 0, expected
						},
					}
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(reader),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateInvalidJSON(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("[")),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if err == nil || !strings.HasSuffix(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateServerBusyFirstCase(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"authorization": ""}`)),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if !errors.Is(err, errServerBusy) {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateServerBusyThirdCase(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{}`)),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if !errors.Is(err, errServerBusy) {
		t.Fatal("not the error we expected")
	}
	if result.Authorization != "" || result.Unchoked != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegotiateSuccess(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader(`{"authorization": "xx", "unchoked": 1}`)),
					}
					return resp, nil
				},
			}
		},
	}

	result, err := negotiate(context.Background(), "", deps)
	if err != nil {
		t.Fatal(err)
	}
	if result.Authorization != "xx" || result.Unchoked != 1 {
		t.Fatal("invalid result")
	}
}
