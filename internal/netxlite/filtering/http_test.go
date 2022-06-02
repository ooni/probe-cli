package filtering

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPProxy(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	httpGET := func(ctx context.Context, URL *url.URL, host string, config *tls.Config) (*http.Response, error) {
		reso := netxlite.NewResolverStdlib(log.Log)
		dialer := netxlite.NewDialerWithResolver(log.Log, reso)
		thx := netxlite.NewTLSHandshakerStdlib(log.Log)
		config = netxlite.ClonedTLSConfigOrNewEmptyConfig(config)
		config.ServerName = host
		tlsDialer := netxlite.NewTLSDialerWithConfig(dialer, thx, config)
		txp := netxlite.NewHTTPTransport(log.Log, dialer, tlsDialer)
		clnt := &http.Client{Transport: txp}
		req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
		runtimex.PanicOnError(err, "http.NewRequest failed")
		req.Host = host
		return clnt.Do(req)
	}

	t.Run("HTTPActionReset", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext(HTTPActionReset)
		resp, err := httpGET(ctx, srvr.URL(), "nexa.polito.it", srvr.TLSConfig())
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureConnectionReset) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		srvr.Close()
	})

	t.Run("HTTPActionTimeout", func(t *testing.T) {
		ctx := context.Background()
		ctx, cancel := context.WithTimeout(ctx, 100*time.Millisecond)
		defer cancel()
		srvr := NewHTTPServerCleartext(HTTPActionTimeout)
		resp, err := httpGET(ctx, srvr.URL(), "nexa.polito.it", srvr.TLSConfig())
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		srvr.Close()
	})

	t.Run("HTTPActionEOF", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext(HTTPActionEOF)
		resp, err := httpGET(ctx, srvr.URL(), "nexa.polito.it", srvr.TLSConfig())
		if err == nil || !strings.HasSuffix(err.Error(), netxlite.FailureEOFError) {
			t.Fatal("unexpected err", err)
		}
		if resp != nil {
			t.Fatal("expected nil resp")
		}
		srvr.Close()
	})

	t.Run("HTTPAction451", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext(HTTPAction451)
		resp, err := httpGET(ctx, srvr.URL(), "nexa.polito.it", srvr.TLSConfig())
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 451 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		data, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(HTTPBlockpage451, data) {
			t.Fatal("unexpected data")
		}
		resp.Body.Close()
		srvr.Close()
	})

	t.Run("HTTPActionDoH", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext(HTTPAction451)
		resp, err := httpGET(ctx, srvr.URL(), "dns.google", srvr.TLSConfig())
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 200 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		data, err := netxlite.ReadAllContext(ctx, resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		response := &dns.Msg{}
		if err := response.Unpack(data); err != nil {
			t.Fatal(err)
		}
		// It suffices to see it's a DNS response
		resp.Body.Close()
		srvr.Close()
	})

	t.Run("unknown action", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext("")
		resp, err := httpGET(ctx, srvr.URL(), "nexa.polito.it", srvr.TLSConfig())
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 500 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		resp.Body.Close()
		srvr.Close()
	})
}
