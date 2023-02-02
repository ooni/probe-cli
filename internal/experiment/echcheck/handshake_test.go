package echcheck

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHandshake(t *testing.T) {
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "success")
	}))
	defer ts.Close()

	parsed, err := url.Parse(ts.URL)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", parsed.Host)
	if err != nil {
		t.Fatal(err)
	}

	result := handshakeWithEch(ctx, conn, time.Now(), parsed.Host, "example.org")
	if result == nil {
		t.Fatal("expected result")
	}

	if result.SoError != nil {
		t.Fatal("did not expect error, got: ", result.SoError)
	}

	if result.Failure != nil {
		t.Fatal("did not expect error, got: ", *result.Failure)
	}

	conn.Close()
}
