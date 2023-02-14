package mlablocatev2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestQueryNDT7Success(t *testing.T) {
	// this integration test is ~0.5 s, so we can always run it

	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")
	result, err := client.QueryNDT7(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) <= 0 {
		t.Fatal("unexpected empty result")
	}

	for _, entry := range result {
		if entry.Hostname == "" {
			t.Fatal("expected non empty Hostname here")
		}
		if entry.Site == "" {
			t.Fatal("expected non-empty Site here")
		}
		if entry.WSSDownloadURL == "" {
			t.Fatal("expected non-empty WSSDownloadURL here")
		}
		if _, err := url.Parse(entry.WSSDownloadURL); err != nil {
			t.Fatal("expected to see a valid URL", err)
		}
		if entry.WSSUploadURL == "" {
			t.Fatal("expected non-empty WSSUploadURL here")
		}
		if _, err := url.Parse(entry.WSSUploadURL); err != nil {
			t.Fatal("expected to see a valid URL", err)
		}
	}
}

func TestQueryDashSuccess(t *testing.T) {
	// this integration test is ~0.5 s, so we can always run it

	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")
	result, err := client.QueryDash(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result) <= 0 {
		t.Fatal("unexpected empty result")
	}

	for _, entry := range result {
		if entry.Hostname == "" {
			t.Fatal("expected non empty Hostname here")
		}
		if entry.Site == "" {
			t.Fatal("expected non-empty Site here")
		}
		if entry.NegotiateURL == "" {
			t.Fatal("expected non-empty NegotiateURL here")
		}
		if _, err := url.Parse(entry.NegotiateURL); err != nil {
			t.Fatal("expected to see a valid URL", err)
		}
		if entry.BaseURL == "" {
			t.Fatal("expected non-empty BaseURL here")
		}
		if _, err := url.Parse(entry.BaseURL); err != nil {
			t.Fatal("expected to see a valid URL", err)
		}
	}
}

func TestQuery404Response(t *testing.T) {
	// this integration test is ~0.5 s, so we can always run it

	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")
	result, err := client.query(context.Background(), "nonexistent")

	if !errors.Is(err, ErrRequestFailed) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected empty results")
	}
}

func TestQueryNewRequestFailure(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	client.Hostname = "\t" // this hostname will cause NewRequest to fail

	result, err := client.query(context.Background(), "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "invalid URL escape") {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryHTTPClientDoFailure(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that fails
	expected := errors.New("mocked error")
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				return nil, expected
			},
		},
	}

	result, err := client.query(context.Background(), "nonexistent")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryCannotReadBody(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that fails
	// when reading the response body
	expected := errors.New("mocked error")
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(&mocks.Reader{
						MockRead: func(b []byte) (int, error) {
							return 0, expected
						},
					}),
				}
				return resp, nil
			},
		},
	}

	result, err := client.query(context.Background(), "nonexistent")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryInvalidJSON(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that returns
	// a non-parsable string
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("{")),
				}
				return resp, nil
			},
		},
	}

	result, err := client.query(context.Background(), "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryNDT7NullResponse(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that returns
	// a literal JSON `null``
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("null")),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryDashNullResponse(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that returns
	// a literal JSON `null``
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("null")),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryDash(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryNDT7EmptyResponse(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that returns
	// an empty JSON object
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryDashEmptyResponse(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client using a mocked client that returns
	// an empty JSON object
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body:       io.NopCloser(strings.NewReader("{}")),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryDash(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryNDT7Fails(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return 404
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 404,
					Body: io.NopCloser(&mocks.Reader{
						MockRead: func(b []byte) (int, error) {
							return 0, io.EOF
						},
					}),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatal("not the error we expected", err)
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryDashFails(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return 404
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 404,
					Body: io.NopCloser(&mocks.Reader{
						MockRead: func(b []byte) (int, error) {
							return 0, io.EOF
						},
					}),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryDash(context.Background())
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatal("not the error we expected", err)
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryNDT7InvalidURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return invalid URLs
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"wss:///ndt/v7/download":":","wss:///ndt/v7/upload":":"}}]}`),
					),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryDashInvalidURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return invalid URLs
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"https:///negotiate/dash":":"}}]}`),
					),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryDash(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryNDT7EmptyURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return empty URLs
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"wss:///ndt/v7/download":"","wss:///ndt/v7/upload":""}}]}`),
					),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestQueryDashEmptyURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, model.DiscardLogger, "miniooni/0.1.0-dev")

	// override the client to return empty URLs
	client.HTTPClient = &http.Client{
		Transport: &mocks.HTTPTransport{
			MockRoundTrip: func(req *http.Request) (*http.Response, error) {
				resp := &http.Response{
					StatusCode: 200,
					Body: io.NopCloser(strings.NewReader(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"https:///negotiate/dash":""}}]}`),
					),
				}
				return resp, nil
			},
		},
	}

	result, err := client.QueryDash(context.Background())
	if !errors.Is(err, ErrEmptyResponse) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

// TestEntryRecordSize is a unit test to make sure we can obtain
// the site name from the returned entries.
func TestEntryRecordSite(t *testing.T) {
	type fields struct {
		Machine string
		URLs    map[string]string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{{
		name: "with invalid machine name",
		fields: fields{
			Machine: "ndt-iupui-mlab3-mil02.mlab-oti.measurement-lab.org",
		},
		want: "",
	}, {
		name: "with valid machine name",
		fields: fields{
			Machine: "mlab3-mil04.mlab-oti.measurement-lab.org",
		},
		want: "mil04",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			er := entryRecord{
				Machine: tt.fields.Machine,
				URLs:    tt.fields.URLs,
			}
			if got := er.Site(); got != tt.want {
				t.Errorf("entryRecord.Site() = %v, want %v", got, tt.want)
			}
		})
	}
}
