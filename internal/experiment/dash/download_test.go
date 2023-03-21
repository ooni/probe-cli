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

func TestDownloadParseURLFailure(t *testing.T) {
	expected := errors.New("mocked error")

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: func(
			ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
			return nil, expected
		},
	}

	_, err := download(context.Background(), downloadConfig{
		deps:    deps,
		baseURL: "\t",
	})

	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadNewHTTPRequestWithContextFailure(t *testing.T) {
	expected := errors.New("mocked error")

	deps := &mockableDependencies{
		MockNewHTTPRequestWithContext: func(
			ctx context.Context, method, url string, body io.Reader) (*http.Request, error) {
			return nil, expected
		},
	}

	_, err := download(context.Background(), downloadConfig{
		deps: deps,
	})

	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadHTTPClientDoFailure(t *testing.T) {
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

	_, err := download(context.Background(), downloadConfig{
		deps: deps,
	})

	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadInternalError(t *testing.T) {

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

	_, err := download(context.Background(), downloadConfig{
		deps: deps,
	})

	if !errors.Is(err, errHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadReadAllFailure(t *testing.T) {
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

	_, err := download(context.Background(), downloadConfig{
		deps: deps,
	})

	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadSuccess(t *testing.T) {

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

	result, err := download(context.Background(), downloadConfig{
		deps: deps,
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
