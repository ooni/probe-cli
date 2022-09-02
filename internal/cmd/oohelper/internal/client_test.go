package internal_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"reflect"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/oohelper/internal"
)

func TestMakeTCPEndpoints(t *testing.T) {
	type args struct {
		URL   *url.URL
		addrs []string
	}
	tests := []struct {
		name string
		args args
		want []string
		err  error
	}{{
		name: "with host != hostname",
		args: args{URL: &url.URL{Host: "127.0.0.1:8080"}},
		err:  internal.ErrUnsupportedExplicitPort,
	}, {
		name: "with unsupported URL scheme",
		args: args{URL: &url.URL{Host: "127.0.0.1", Scheme: "imap"}},
		err:  internal.ErrUnsupportedURLScheme,
	}, {
		name: "with http scheme",
		args: args{
			URL:   &url.URL{Host: "www.kernel.org", Scheme: "http"},
			addrs: []string{"1.1.1.1", "2.2.2.2", "::1"},
		},
		want: []string{"1.1.1.1:80", "2.2.2.2:80", "[::1]:80"},
	}, {
		name: "with https scheme",
		args: args{
			URL:   &url.URL{Host: "www.kernel.org", Scheme: "https"},
			addrs: []string{"1.1.1.1", "2.2.2.2", "::1"},
		},
		want: []string{"1.1.1.1:443", "2.2.2.2:443", "[::1]:443"},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := internal.MakeTCPEndpoints(tt.args.URL, tt.args.addrs)
			if !errors.Is(err, tt.err) {
				t.Errorf("MakeTCPEndpoints() error = %v, wantErr %v", err, tt.err)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("MakeTCPEndpoints() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestOOClientDoWithEmptyTargetURL(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{}
	clnt := internal.OOClient{}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrEmptyURL) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithEmptyServerURL(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{TargetURL: "http://www.example.com"}
	clnt := internal.OOClient{}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrEmptyURL) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithInvalidTargetURL(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{TargetURL: "\t", ServerURL: "https://0.th.ooni.org"}
	clnt := internal.OOClient{}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrInvalidURL) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithResolverFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		HTTPClient: http.DefaultClient,
		Resolver:   internal.NewFakeResolverThatFails(),
	}
	cresp, err := clnt.Do(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(cresp.TCPConnect) <= 0 {
		// The legacy implementation of the test helper (the legacy codebase)
		// only follows the IP addresses returned by the client. However, since
		// https://github.com/ooni/probe-cli/pull/890, the TH is following the
		// IP addresses from the probe as well as its own addresses.
		t.Fatal("expected non-empty TCPConnect here")
	}
	if cresp.HTTPRequest.StatusCode != 200 {
		t.Fatal("expected 200 status code here")
	}
	if len(cresp.DNS.Addrs) < 1 {
		t.Fatal("expected at least an IP address here")
	}
}

func TestOOClientDoWithUnsupportedExplicitPort(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com:8080",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrUnsupportedExplicitPort) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithInvalidServerURL(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "\t",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrCannotCreateRequest) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithRoundTripError(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
		HTTPClient: &http.Client{
			Transport: internal.FakeTransport{Err: expected},
		},
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithInvalidStatusCode(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
		HTTPClient: &http.Client{Transport: internal.FakeTransport{
			Resp: &http.Response{
				StatusCode: 400,
				Body:       &internal.FakeBody{},
			},
		}},
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrHTTPStatusCode) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithBodyReadError(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
		HTTPClient: &http.Client{Transport: internal.FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: &internal.FakeBody{
					Err: expected,
				},
			},
		}},
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

func TestOOClientDoWithInvalidJSON(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
		HTTPClient: &http.Client{Transport: internal.FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: &internal.FakeBody{
					Data: []byte("{"),
				},
			},
		}},
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, internal.ErrCannotParseJSONReply) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp != nil {
		t.Fatal("expected nil response")
	}
}

const goodresponse = `{
    "tcp_connect": {
        "172.217.21.68:80": {
            "status": true,
            "failure": null
        }
    },
    "http_request": {
        "body_length": 207878,
        "failure": null,
        "title": "Google",
        "headers": {
            "Content-Type": "text/html"
        },
        "status_code": 200
    },
    "dns": {
        "failure": null,
        "addrs": [
            "172.217.17.68"
        ]
    }
}`

func TestOOClientDoWithParseableJSON(t *testing.T) {
	ctx := context.Background()
	config := internal.OOConfig{
		TargetURL: "http://www.example.com",
		ServerURL: "https://0.th.ooni.org",
	}
	clnt := internal.OOClient{
		Resolver: internal.NewFakeResolverWithResult([]string{"1.1.1.1"}),
		HTTPClient: &http.Client{Transport: internal.FakeTransport{
			Resp: &http.Response{
				StatusCode: 200,
				Body: &internal.FakeBody{
					Data: []byte(goodresponse),
				},
			},
		}},
	}
	cresp, err := clnt.Do(ctx, config)
	if !errors.Is(err, nil) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if cresp.DNS.Failure != nil {
		t.Fatal("unexpected Failure value")
	}
	if len(cresp.DNS.Addrs) != 1 {
		t.Fatal("unexpected number of DNS entries")
	}
	if cresp.DNS.Addrs[0] != "172.217.17.68" {
		t.Fatal("unexpected DNS addrs [0]")
	}
	if cresp.HTTPRequest.BodyLength != 207878 {
		t.Fatal("invalid http body length")
	}
	if cresp.HTTPRequest.Failure != nil {
		t.Fatal("invalid http failure")
	}
	if cresp.HTTPRequest.Title != "Google" {
		t.Fatal("invalid http title")
	}
	if len(cresp.HTTPRequest.Headers) != 1 {
		t.Fatal("invalid http headers length")
	}
	if cresp.HTTPRequest.Headers["Content-Type"] != "text/html" {
		t.Fatal("invalid http content-type header")
	}
	if cresp.HTTPRequest.StatusCode != 200 {
		t.Fatal("invalid http status code")
	}
	if len(cresp.TCPConnect) != 1 {
		t.Fatal("invalid tcp connect length")
	}
	entry, ok := cresp.TCPConnect["172.217.21.68:80"]
	if !ok {
		t.Fatal("cannot find expected TCP connect entry")
	}
	if entry.Status != true {
		t.Fatal("unexpected TCP connect entry status")
	}
	if entry.Failure != nil {
		t.Fatal("unexpected TCP connect entry failure value")
	}
}
