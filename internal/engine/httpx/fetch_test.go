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
