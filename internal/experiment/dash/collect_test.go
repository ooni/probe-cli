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

func TestCollectNewHTTPRequestWithContextFailure(t *testing.T) {
	expected := errors.New("mocked error")

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: func(
			ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
			return nil, expected
		},
	}

	err := collect(context.Background(), "", "", nil, deps)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestCollectHTTPClientDoFailure(t *testing.T) {
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

	err := collect(context.Background(), "", "", nil, deps)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestCollectInternalError(t *testing.T) {

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

	err := collect(context.Background(), "", "", nil, deps)
	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
}

func TestCollectReadAllFailure(t *testing.T) {
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

	err := collect(context.Background(), "", "", nil, deps)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestCollectInvalidJSON(t *testing.T) {

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

	err := collect(context.Background(), "", "", nil, deps)
	if err == nil || !strings.HasSuffix(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expected")
	}
}

func TestCollectSuccess(t *testing.T) {

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: http.NewRequestWithContext,
		MockHTTPClient: func() model.HTTPClient {
			return &mocks.HTTPClient{
				MockDo: func(req *http.Request) (*http.Response, error) {
					resp := &http.Response{
						StatusCode: 200,
						Body:       io.NopCloser(strings.NewReader("[]")),
					}
					return resp, nil
				},
			}
		},
	}

	err := collect(context.Background(), "", "", nil, deps)
	if err != nil {
		t.Fatal(err)
	}
}
