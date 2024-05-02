package httpclientx

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestGzipDecompression(t *testing.T) {
	t.Run("we correctly handle gzip encoding", func(t *testing.T) {
		expected := []byte(`Bonsoir, Elliot!!!`)

		// create a server returning compressed content
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var buffer bytes.Buffer
			writer := gzip.NewWriter(&buffer)
			_ = runtimex.Try1(writer.Write(expected))
			runtimex.Try0(writer.Close())
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(buffer.Bytes())
		}))
		defer server.Close()

		// make sure we can read it
		respbody, err := GetRaw(
			context.Background(),
			NewEndpoint(server.URL),
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			})

		t.Log(respbody)
		t.Log(err)

		if err != nil {
			t.Fatal(err)
		}

		if diff := cmp.Diff(expected, respbody); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("we correctly handle the case where we cannot decode gzip", func(t *testing.T) {
		expected := []byte(`Bonsoir, Elliot!!!`)

		// create a server pretending to return compressed content
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("Content-Encoding", "gzip")
			w.Write(expected)
		}))
		defer server.Close()

		// attempt to get a response body
		respbody, err := GetRaw(
			context.Background(),
			NewEndpoint(server.URL),
			&Config{
				Client:    http.DefaultClient,
				Logger:    model.DiscardLogger,
				UserAgent: model.HTTPHeaderUserAgent,
			})

		t.Log(respbody)
		t.Log(err)

		if err.Error() != "gzip: invalid header" {
			t.Fatal(err)
		}

		if respbody != nil {
			t.Fatal("expected nil response body")
		}
	})
}

func TestHTTPStatusCodeHandling(t *testing.T) {
	server := testingx.MustNewHTTPServer(testingx.HTTPHandlerBlockpage451())
	defer server.Close()

	respbody, err := GetRaw(
		context.Background(),
		NewEndpoint(server.URL),
		&Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

	t.Log(respbody)
	t.Log(err)

	if err.Error() != "httpx: request failed" {
		t.Fatal(err)
	}

	if respbody != nil {
		t.Fatal("expected nil response body")
	}

	var orig *ErrRequestFailed
	if !errors.As(err, &orig) {
		t.Fatal("not an *ErrRequestFailed instance")
	}
	if orig.StatusCode != 451 {
		t.Fatal("unexpected status code", orig.StatusCode)
	}
}

func TestHTTPReadBodyErrorsHandling(t *testing.T) {
	server := testingx.MustNewHTTPServer(testingx.HTTPHandlerResetWhileReadingBody())
	defer server.Close()

	respbody, err := GetRaw(
		context.Background(),
		NewEndpoint(server.URL),
		&Config{
			Client:    http.DefaultClient,
			Logger:    model.DiscardLogger,
			UserAgent: model.HTTPHeaderUserAgent,
		})

	t.Log(respbody)
	t.Log(err)

	if !errors.Is(err, netxlite.ECONNRESET) {
		t.Fatal("expected ECONNRESET, got", err)
	}

	if respbody != nil {
		t.Fatal("expected nil response body")
	}
}
