package urlgetter_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
)

func TestRunnerWithInvalidURLScheme(t *testing.T) {
	r := urlgetter.Runner{Target: "antani://www.google.com"}
	err := r.Run(context.Background())
	if err == nil || err.Error() != "unknown targetURL scheme" {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerHTTPWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := urlgetter.Runner{Target: "https://www.google.com"}
	err := r.Run(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerDNSLookupWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := urlgetter.Runner{Target: "dnslookup://www.google.com"}
	err := r.Run(ctx)
	if err == nil || err.Error() != "interrupted" {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerTLSHandshakeWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := urlgetter.Runner{Target: "tlshandshake://www.google.com:443"}
	err := r.Run(ctx)
	if err == nil || err.Error() != "interrupted" {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerTCPConnectWithContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	r := urlgetter.Runner{Target: "tcpconnect://www.google.com:443"}
	err := r.Run(ctx)
	if err == nil || err.Error() != "interrupted" {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerWithInvalidURL(t *testing.T) {
	r := urlgetter.Runner{Target: "\t"}
	err := r.Run(context.Background())
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerWithEmptyHostname(t *testing.T) {
	r := urlgetter.Runner{Target: "http:///foo.txt"}
	err := r.Run(context.Background())
	if err == nil || !strings.HasSuffix(err.Error(), "no Host in request URL") {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerTLSHandshakeSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	r := urlgetter.Runner{Target: "tlshandshake://www.google.com:443"}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunnerTCPConnectSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	r := urlgetter.Runner{Target: "tcpconnect://www.google.com:443"}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunnerDNSLookupSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	r := urlgetter.Runner{Target: "dnslookup://www.google.com"}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunnerHTTPSSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	r := urlgetter.Runner{Target: "https://www.google.com"}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunnerHTTPSetHostHeader(t *testing.T) {
	var host string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host = r.Host
		w.WriteHeader(200)
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			HTTPHost: "x.org",
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if host != "x.org" {
		t.Fatal("not the host we expected")
	}
}

func TestRunnerHTTPNoRedirect(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Location", "http:///") // cause failure if we redirect
		w.WriteHeader(302)
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
}

func TestRunnerHTTPCannotReadBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			panic("hijacking not supported by this server")
		}
		conn, _, _ := hijacker.Hijack()
		conn.Write([]byte("HTTP/1.1 200 Ok\r\n"))
		conn.Write([]byte("Content-Length: 1024\r\n"))
		conn.Write([]byte("\r\n"))
		conn.Write([]byte("123456789"))
		conn.Close()
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerHTTPWeHandle400Correctly(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			FailOnHTTPError:   true,
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if !errors.Is(err, urlgetter.ErrHTTPRequestFailed) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerHTTPCannotReadBodyWinsOver400(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			panic("hijacking not supported by this server")
		}
		conn, _, _ := hijacker.Hijack()
		conn.Write([]byte("HTTP/1.1 400 Bad Request\r\n"))
		conn.Write([]byte("Content-Length: 1024\r\n"))
		conn.Write([]byte("\r\n"))
		conn.Write([]byte("123456789"))
		conn.Close()
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			FailOnHTTPError:   true,
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerWeCanForceUserAgent(t *testing.T) {
	expected := "antani/1.23.4-dev"
	found := atomicx.NewInt64()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == expected {
			found.Add(1)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			FailOnHTTPError:   true,
			NoFollowRedirects: true,
			UserAgent:         expected,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if found.Load() != 1 {
		t.Fatal("we didn't override the user agent")
	}
}

func TestRunnerDefaultUserAgent(t *testing.T) {
	expected := httpheader.UserAgent()
	found := atomicx.NewInt64()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") == expected {
			found.Add(1)
		}
		w.WriteHeader(200)
	}))
	defer server.Close()
	r := urlgetter.Runner{
		Config: urlgetter.Config{
			FailOnHTTPError:   true,
			NoFollowRedirects: true,
		},
		Target: server.URL,
	}
	err := r.Run(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if found.Load() != 1 {
		t.Fatal("we didn't override the user agent")
	}
}
