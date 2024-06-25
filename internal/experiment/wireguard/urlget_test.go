package wireguard

import (
	"context"
	"errors"
	"math"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

type failingHttpClient struct{}

func (c *failingHttpClient) Get(string) (*http.Response, error) {
	return nil, errors.New("some error")
}

func Test_urlget(t *testing.T) {
	t.Run("dummy server gets a URLGetResult, with no error", func(t *testing.T) {
		expected := "dummy data"
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte(expected))
		}))
		defer svr.Close()

		m := &Measurer{}
		m.dialContextFn = func(_ context.Context, network, address string) (net.Conn, error) {
			return net.Dial(network, address)
		}
		r := m.urlget(svr.URL, time.Now(), model.DiscardLogger)
		if r.StatusCode != 200 {
			t.Fatal("expected statusCode==200")
		}
	})

	t.Run("dummy server gets a URLGetResult with 500 status code", func(t *testing.T) {
		expected := "dummy data"
		svr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(500)
			w.Write([]byte(expected))
		}))
		defer svr.Close()

		m := &Measurer{}
		m.dialContextFn = func(_ context.Context, network, address string) (net.Conn, error) {
			return net.Dial(network, address)
		}
		r := m.urlget(svr.URL, time.Now(), model.DiscardLogger)
		if r.StatusCode != 500 {
			t.Fatal("expected statusCode==500")
		}
	})

	t.Run("client returns error", func(t *testing.T) {
		m := &Measurer{}
		m.httpClient = &failingHttpClient{}

		r := m.urlget("http://example.org", time.Now(), model.DiscardLogger)
		expectedError := "unknown_failure: some error"
		if *r.Failure != expectedError {
			t.Fatal("expected error")
		}
	})
}

func Test_newURLResultFromError(t *testing.T) {
	url := "https://example.org"
	zeroTime := time.Now().Add(-1 * time.Second)
	start := 0.1
	err := errors.New("some error")

	r := newURLResultFromError(url, zeroTime, start, err)
	if r.URL != url {
		t.Fatal("wrong url")
	}
	if r.T0 != start {
		t.Fatal("wrong t0")
	}
	if math.Abs(r.T-1.0) > 0.01 {
		t.Fatal("should be ~now, not", r.T)
	}
	if r.Error != err.Error() {
		t.Fatal("wrong error")
	}
	expectedFailure := "unknown_failure: " + err.Error()
	if *r.Failure != expectedFailure {
		t.Fatal(*r.Failure)
	}
}

func Test_newURLResultWithStratusCode(t *testing.T) {
	url := "https://example.org"
	zeroTime := time.Now().Add(-1 * time.Second)
	start := 0.1

	r := newURLResultWithStatusCode(url, zeroTime, start, 200, []byte("potatoes"))
	if r.URL != url {
		t.Fatal("wrong url")
	}
	if r.T0 != start {
		t.Fatal("wrong t0")
	}
	if math.Abs(r.T-1.0) > 0.01 {
		t.Fatal("should be ~now, not", r.T)
	}
	if r.StatusCode != 200 {
		t.Fatal("expected statusCode==200")
	}
	if r.ByteCount != 8 {
		t.Fatal("expected byteCount=8")
	}
}
