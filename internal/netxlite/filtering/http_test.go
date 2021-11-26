package filtering

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPProxy(t *testing.T) {
	newproxy := func(action HTTPAction) (net.Listener, error) {
		p := &HTTPProxy{
			OnIncomingHost: func(host string) HTTPAction {
				return action
			},
		}
		return p.Start("127.0.0.1:0")
	}

	httpGET := func(ctx context.Context, addr net.Addr, host string) (*http.Response, error) {
		txp := netxlite.NewHTTPTransportStdlib(log.Log)
		clnt := &http.Client{Transport: txp}
		URL := &url.URL{
			Scheme: "http",
			Host:   addr.String(),
			Path:   "/",
		}
		req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
		runtimex.PanicOnError(err, "http.NewRequest failed")
		req.Host = host
		return clnt.Do(req)
	}

	t.Run("HTTPActionPass", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy(HTTPActionPass)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		resp.Body.Close()
		listener.Close()
	})

	t.Run("HTTPActionPass with self connect", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy(HTTPActionPass)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), listener.Addr().String())
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 400 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		resp.Body.Close()
		listener.Close()
	})

	t.Run("HTTPActionReset", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy(HTTPActionReset)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureConnectionReset) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		listener.Close()
	})

	t.Run("HTTPActionTimeout", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		listener, err := newproxy(HTTPActionTimeout)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		listener.Close()
	})

	t.Run("HTTPActionEOF", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy(HTTPActionEOF)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureEOFError) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		listener.Close()
	})

	t.Run("HTTPAction451", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy(HTTPAction451)
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 451 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		resp.Body.Close()
		listener.Close()
	})

	t.Run("unknown action", func(t *testing.T) {
		ctx := context.Background()
		listener, err := newproxy("")
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, listener.Addr(), "nexa.polito.it")
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 500 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		resp.Body.Close()
		listener.Close()
	})

	t.Run("Start fails on an invalid address", func(t *testing.T) {
		p := &HTTPProxy{}
		listener, err := p.Start("127.0.0.1")
		if err == nil || !strings.HasSuffix(err.Error(), "missing port in address") {
			t.Fatal("unexpected err", err)
		}
		if listener != nil {
			t.Fatal("expected nil listener")
		}
	})
}
