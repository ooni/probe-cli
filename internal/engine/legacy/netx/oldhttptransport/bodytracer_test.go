package oldhttptransport

import (
	"context"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/iox"
)

func TestBodyTracerSuccess(t *testing.T) {
	client := &http.Client{
		Transport: NewBodyTracer(http.DefaultTransport),
	}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	_, err = iox.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	client.CloseIdleConnections()
}

func TestBodyTracerFailure(t *testing.T) {
	client := &http.Client{
		Transport: NewBodyTracer(http.DefaultTransport),
	}
	// This fails the request because we attempt to speak cleartext HTTP with
	// a server that instead is expecting TLS.
	resp, err := client.Get("http://www.google.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if resp != nil {
		t.Fatal("expected a nil response here")
	}
	client.CloseIdleConnections()
}
