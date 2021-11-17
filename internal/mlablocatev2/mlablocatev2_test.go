package mlablocatev2

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
)

func TestSuccess(t *testing.T) {
	// this test is ~0.5 s, so we can always run it
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
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
			t.Fatal("expected non=-empty Site here")
		}
		if entry.WSSDownloadURL == "" {
			t.Fatal("expected non-empty WSSDownloadURL here")
		}
		if _, err := url.Parse(entry.WSSDownloadURL); err != nil {
			t.Fatal(err)
		}
		if entry.WSSUploadURL == "" {
			t.Fatal("expected non-empty WSSUploadURL here")
		}
		if _, err := url.Parse(entry.WSSUploadURL); err != nil {
			t.Fatal(err)
		}
	}
}

func Test404Response(t *testing.T) {
	// this test is ~0.5 s, so we can always run it
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	result, err := client.query(context.Background(), "nonexistent")
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatal("not the error we expected")
	}
	if result.Results != nil {
		t.Fatal("expected empty results")
	}
}

func TestNewRequestFailure(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.Hostname = "\t"
	result, err := client.query(context.Background(), "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "invalid URL escape") {
		t.Fatal("not the error we expected")
	}
	if result.Results != nil {
		t.Fatal("expected nil results")
	}
}

func TestHTTPClientDoFailure(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	expected := errors.New("mocked error")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{Err: expected},
	}
	result, err := client.query(context.Background(), "nonexistent")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Results != nil {
		t.Fatal("expected nil results")
	}
}

func TestCannotReadBody(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	expected := errors.New("mocked error")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: FakeBody{
					Err: expected,
				},
			},
		},
	}
	result, err := client.query(context.Background(), "nonexistent")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if result.Results != nil {
		t.Fatal("expected nil results")
	}
}

func TestInvalidJSON(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: FakeBody{
					Err:  io.EOF,
					Data: []byte(`{`),
				},
			},
		},
	}
	result, err := client.query(context.Background(), "nonexistent")
	if err == nil || !strings.Contains(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expected")
	}
	if result.Results != nil {
		t.Fatal("expected nil results")
	}
}

func TestEmptyResponse(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: FakeBody{
					Err:  io.EOF,
					Data: []byte(`{}`),
				},
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

func TestNDT7QueryFails(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 404,
				Body:       FakeBody{Err: io.EOF},
			},
		},
	}
	result, err := client.QueryNDT7(context.Background())
	if !errors.Is(err, ErrRequestFailed) {
		t.Fatal("not the error we expected")
	}
	if result != nil {
		t.Fatal("expected nil results")
	}
}

func TestNDT7InvalidURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: FakeBody{
					Data: []byte(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"wss:///ndt/v7/download":":","wss:///ndt/v7/upload":":"}}]}`),
					Err: io.EOF,
				},
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

func TestNDT7EmptyURLs(t *testing.T) {
	client := NewClient(http.DefaultClient, log.Log, "miniooni/0.1.0-dev")
	client.HTTPClient = &http.Client{
		Transport: FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: FakeBody{
					Data: []byte(
						`{"results":[{"machine":"mlab3-mil04.mlab-oti.measurement-lab.org","urls":{"wss:///ndt/v7/download":"","wss:///ndt/v7/upload":""}}]}`),
					Err: io.EOF,
				},
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
