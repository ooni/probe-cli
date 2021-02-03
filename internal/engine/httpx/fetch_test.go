package httpx_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
)

func TestFetchResourceIntegration(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (httpx.Client{
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
	data, err := (httpx.Client{
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

func TestFetchResourceAndVerifyIntegration(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (httpx.Client{
		BaseURL:    "https://github.com/",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResourceAndVerify(
		ctx,
		"/measurement-kit/generic-assets/releases/download/20190426155936/generic-assets-20190426155936.tar.gz",
		"34d8a9c8ab30c242469482dc280be832d8a06b4400f8927604dd361bf979b795",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) <= 0 {
		t.Fatal("Did not expect an empty resource")
	}
}

func TestFetchResourceInvalidURL(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (httpx.Client{
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
	data, err := (httpx.Client{
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

func TestFetchResourceAndVerify400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(400)
		},
	))
	defer server.Close()
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (httpx.Client{
		BaseURL:    server.URL,
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResourceAndVerify(ctx, "", "abcde")
	if err == nil || !strings.HasSuffix(err.Error(), "400 Bad Request") {
		t.Fatal("not the error we expected")
	}
	if len(data) != 0 {
		t.Fatal("expected an empty resource")
	}
}

func TestFetchResourceAndVerifyInvalidSHA256(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	ctx := context.Background()
	data, err := (httpx.Client{
		BaseURL:    "https://github.com/",
		HTTPClient: http.DefaultClient,
		Logger:     log.Log,
		UserAgent:  "ooniprobe-engine/0.1.0",
	}).FetchResourceAndVerify(
		ctx,
		"/measurement-kit/generic-assets/releases/download/20190426155936/generic-assets-20190426155936.tar.gz",
		"34d8a9ceeb30c242469482dc280be832d8a06b4400f8927604dd361bf979b795",
	)
	if err == nil || !strings.HasPrefix(err.Error(), "httpx: SHA256 mismatch:") {
		t.Fatal("not the error we expected")
	}
	if len(data) != 0 {
		t.Fatal("expected an empty resource")
	}
}
