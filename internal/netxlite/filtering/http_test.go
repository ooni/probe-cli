package filtering

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestHTTPServer(t *testing.T) {

	httpGET := func(ctx context.Context, method string, URL *url.URL, host string,
		config *tls.Config, requestBody []byte) (*http.Response, error) {
		txp := &http.Transport{
			TLSClientConfig: config,
		}
		if config != nil {
			config.ServerName = host
		}
		clnt := &http.Client{Transport: txp}
		req, err := http.NewRequestWithContext(
			ctx, method, URL.String(), bytes.NewReader(requestBody))
		runtimex.PanicOnError(err, "http.NewRequest failed")
		req.Host = host
		return clnt.Do(req)
	}

	t.Run("HTTPActionReset", func(t *testing.T) {
		ctx := context.Background()
		srvr := NewHTTPServerCleartext(HTTPActionReset)
		resp, err := httpGET(ctx, "GET", srvr.URL(), "nexa.polito.it", srvr.TLSConfig(), nil)
		if netxlite.NewTopLevelGenericErrWrapper(err).Error() != netxlite.FailureConnectionReset {
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
		resp, err := httpGET(ctx, "GET", srvr.URL(), "nexa.polito.it", srvr.TLSConfig(), nil)
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
		resp, err := httpGET(ctx, "GET", srvr.URL(), "nexa.polito.it", srvr.TLSConfig(), nil)
		if !errors.Is(err, io.EOF) {
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
		resp, err := httpGET(ctx, "GET", srvr.URL(), "nexa.polito.it", srvr.TLSConfig(), nil)
		if err != nil {
			t.Fatal(err)
		}
		if resp.StatusCode != 451 {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		data, err := netxlite.ReadAllContext(ctx, resp.Body)
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
		srvr := NewHTTPServerTLS(HTTPActionDoH)
		query := dnsComposeQuery("nexa.polito.it", dns.TypeA)
		rawQuery, err := query.Pack()
		if err != nil {
			t.Fatal(err)
		}
		resp, err := httpGET(ctx, "POST", srvr.URL(), "dns.google", srvr.TLSConfig(), rawQuery)
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
		resp, err := httpGET(ctx, "GET", srvr.URL(), "nexa.polito.it", srvr.TLSConfig(), nil)
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

type httpResponseWriter struct {
	http.ResponseWriter
	code int
}

func (w *httpResponseWriter) WriteHeader(statusCode int) {
	w.code = statusCode
}

func TestHTTPServeDNSOverHTTPSPanic(t *testing.T) {
	w := &httpResponseWriter{}
	req := &http.Request{
		Body: io.NopCloser(&mocks.Reader{
			MockRead: func(b []byte) (int, error) {
				return 0, io.ErrUnexpectedEOF
			},
		}),
	}
	httpServeDNSOverHTTPS(w, req)
	if w.code != 500 {
		t.Fatal("did not intercept the panic")
	}
}
