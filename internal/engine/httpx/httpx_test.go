package httpx

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
)

const userAgent = "miniooni/0.1.0-dev"

func newClient() Client {
	return Client{
		BaseURL:    "https://httpbin.org",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  userAgent,
	}
}

func TestNewRequestWithJSONBodyJSONMarshalFailure(t *testing.T) {
	client := newClient()
	req, err := client.NewRequestWithJSONBody(
		context.Background(), "GET", "/", nil, make(chan interface{}),
	)
	if err == nil || !strings.HasPrefix(err.Error(), "json: unsupported type") {
		t.Fatal("not the error we expected")
	}
	if req != nil {
		t.Fatal("expected nil request here")
	}
}

func TestNewRequestWithJSONBodyNewRequestFailure(t *testing.T) {
	client := newClient()
	client.BaseURL = "\t\t\t" // cause URL parse error
	req, err := client.NewRequestWithJSONBody(
		context.Background(), "GET", "/", nil, nil,
	)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if req != nil {
		t.Fatal("expected nil request here")
	}
}

func TestNewRequestWithQuery(t *testing.T) {
	client := newClient()
	q := url.Values{}
	q.Add("antani", "mascetti")
	q.Add("melandri", "conte")
	req, err := client.NewRequest(
		context.Background(), "GET", "/", q, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.URL.Query().Get("antani") != "mascetti" {
		t.Fatal("expected different query string here")
	}
	if req.URL.Query().Get("melandri") != "conte" {
		t.Fatal("expected different query string here")
	}
}

func TestNewRequestNewRequestFailure(t *testing.T) {
	client := newClient()
	req, err := client.NewRequest(
		context.Background(), "\t\t\t", "/", nil, nil,
	)
	if err == nil || !strings.HasPrefix(err.Error(), "net/http: invalid method") {
		t.Fatal("not the error we expected")
	}
	if req != nil {
		t.Fatal("expected nil request here")
	}
}

func TestNewRequestCloudfronting(t *testing.T) {
	client := newClient()
	client.Host = "www.x.org"
	req, err := client.NewRequest(
		context.Background(), "GET", "/", nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.Host != client.Host {
		t.Fatal("expected different req.Host here")
	}
}

func TestNewRequestAcceptIsSet(t *testing.T) {
	client := newClient()
	client.Accept = "application/xml"
	req, err := client.NewRequestWithJSONBody(
		context.Background(), "GET", "/", nil, []string{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Accept") != "application/xml" {
		t.Fatal("expected different Accept here")
	}
}

func TestNewRequestContentTypeIsSet(t *testing.T) {
	client := newClient()
	req, err := client.NewRequestWithJSONBody(
		context.Background(), "GET", "/", nil, []string{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Content-Type") != "application/json" {
		t.Fatal("expected different Content-Type here")
	}
}

func TestNewRequestAuthorizationHeader(t *testing.T) {
	client := newClient()
	client.Authorization = "deadbeef"
	req, err := client.NewRequest(
		context.Background(), "GET", "/", nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("Authorization") != client.Authorization {
		t.Fatal("expected different Authorization here")
	}
}

func TestNewRequestUserAgentIsSet(t *testing.T) {
	client := newClient()
	req, err := client.NewRequest(
		context.Background(), "GET", "/", nil, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if req.Header.Get("User-Agent") != userAgent {
		t.Fatal("expected different User-Agent here")
	}
}

func TestClientDoJSONClientDoFailure(t *testing.T) {
	expected := errors.New("mocked error")
	client := newClient()
	client.HTTPClient = &http.Client{Transport: FakeTransport{
		Err: expected,
	}}
	err := client.DoJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestClientDoJSONResponseNotSuccessful(t *testing.T) {
	client := newClient()
	client.HTTPClient = &http.Client{Transport: FakeTransport{
		Resp: &http.Response{
			StatusCode: 401,
			Body:       FakeBody{},
		},
	}}
	err := client.DoJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
	if err == nil || !strings.HasPrefix(err.Error(), "httpx: request failed") {
		t.Fatal("not the error we expected")
	}
}

func TestClientDoJSONResponseReadingBodyError(t *testing.T) {
	expected := errors.New("mocked error")
	client := newClient()
	client.HTTPClient = &http.Client{Transport: FakeTransport{
		Resp: &http.Response{
			StatusCode: 200,
			Body: FakeBody{
				Err: expected,
			},
		},
	}}
	err := client.DoJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestClientDoJSONResponseIsNotJSON(t *testing.T) {
	client := newClient()
	client.HTTPClient = &http.Client{Transport: FakeTransport{
		Resp: &http.Response{
			StatusCode: 200,
			Body: FakeBody{
				Err: io.EOF,
			},
		},
	}}
	err := client.DoJSON(&http.Request{URL: &url.URL{Scheme: "https", Host: "x.org"}}, nil)
	if err == nil || err.Error() != "unexpected end of JSON input" {
		t.Fatal("not the error we expected")
	}
}

type httpbinheaders struct {
	Headers map[string]string `json:"headers"`
}

func TestReadJSONSuccess(t *testing.T) {
	var headers httpbinheaders
	err := newClient().GetJSON(context.Background(), "/headers", &headers)
	if err != nil {
		t.Fatal(err)
	}
	if headers.Headers["Host"] != "httpbin.org" {
		t.Fatal("unexpected Host header")
	}
	if headers.Headers["User-Agent"] != "miniooni/0.1.0-dev" {
		t.Fatal("unexpected Host header")
	}
}

type httpbinpost struct {
	Data string `json:"data"`
}

func TestCreateJSONSuccess(t *testing.T) {
	headers := httpbinheaders{
		Headers: map[string]string{
			"Foo": "bar",
		},
	}
	var response httpbinpost
	err := newClient().PostJSON(context.Background(), "/post", &headers, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Data != `{"headers":{"Foo":"bar"}}` {
		t.Fatal(response.Data)
	}
}

func TestUpdateJSONSuccess(t *testing.T) {
	headers := httpbinheaders{
		Headers: map[string]string{
			"Foo": "bar",
		},
	}
	var response httpbinpost
	err := newClient().PutJSON(context.Background(), "/put", &headers, &response)
	if err != nil {
		t.Fatal(err)
	}
	if response.Data != `{"headers":{"Foo":"bar"}}` {
		t.Fatal(response.Data)
	}
}

func TestReadJSONFailure(t *testing.T) {
	var headers httpbinheaders
	client := newClient()
	client.BaseURL = "\t\t\t\t"
	err := client.GetJSON(context.Background(), "/headers", &headers)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestCreateJSONFailure(t *testing.T) {
	var headers httpbinheaders
	client := newClient()
	client.BaseURL = "\t\t\t\t"
	err := client.PostJSON(context.Background(), "/headers", &headers, &headers)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestUpdateJSONFailure(t *testing.T) {
	var headers httpbinheaders
	client := newClient()
	client.BaseURL = "\t\t\t\t"
	err := client.PutJSON(context.Background(), "/headers", &headers, &headers)
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestFetchResourceIntegration(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (Client{
		BaseURL:    "http://facebook.com/",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResource(ctx, "/robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 0 {
		t.Fatal("Did not expect an empty resource")
	}
}

func TestFetchResourceExpiredContext(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	data, err := (Client{
		BaseURL:    "http://facebook.com/",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResource(ctx, "/robots.txt")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if len(data) != 0 {
		t.Fatal("expected an empty resource")
	}
}

func TestFetchResourceInvalidURL(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (Client{
		BaseURL:    "http://\t/",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResource(ctx, "/robots.txt")
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if len(data) != 0 {
		t.Fatal("expected an empty resource")
	}
}

func TestFetchResource400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
		},
	))
	defer server.Close()
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (Client{
		Authorization: "foobar",
		BaseURL:       server.URL,
		HTTPClient:    http.DefaultClient,
		Logger:        log.Log,
		UserAgent:     "ooniprobe-engine/0.1.0",
	}).FetchResource(ctx, "")
	if err == nil || !strings.HasSuffix(err.Error(), "400 Bad Request") {
		t.Fatal("not the error we expected")
	}
	if len(data) != 0 {
		t.Fatal("expected an empty resource")
	}
}
