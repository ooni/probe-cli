package browserhttp

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/browserhttp/browserhttptest"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestTransportFingerprint(t *testing.T) {
	for _, proto := range []string{"h2", "http/1.1"} {
		t.Run(proto, func(t *testing.T) {
			srv := browserhttptest.NewServer(proto)
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			conn, err := net.Dial("tcp", srv.Addr())
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			netx := &netxlite.Netx{}
			tlsConn, err := netx.NewTLSHandshakerUTLS(model.DiscardLogger, ClientHello).Handshake(
				ctx, conn, &tls.Config{ServerName: "www.example.com", RootCAs: srv.CertPool})
			if err != nil {
				t.Fatal(err)
			}
			req, err := http.NewRequestWithContext(ctx, "GET", "https://www.example.com/", nil)
			if err != nil {
				t.Fatal(err)
			}
			SetHeaders(req.Header, true)
			alpn := netxlite.MaybeTLSConnectionState(tlsConn).NegotiatedProtocol
			body := fetch(t, NewTransport(tlsConn, alpn), req)

			fp, err := srv.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			browserhttptest.Check(t, proto, fp, body)
		})
	}
}

func TestDialingTransportFingerprint(t *testing.T) {
	for _, proto := range []string{"h2", "http/1.1"} {
		t.Run(proto, func(t *testing.T) {
			srv := browserhttptest.NewServer(proto)
			defer srv.Close()
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			// www.example.com resolves to the server
			reso := &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"127.0.0.1"}, nil
				},
				MockNetwork:              func() string { return "mock" },
				MockAddress:              func() string { return "" },
				MockCloseIdleConnections: func() {},
			}
			netx := &netxlite.Netx{}
			txp := &DialingTransport{
				Dialer:     netx.NewDialerWithResolver(model.DiscardLogger, reso),
				Handshaker: netx.NewTLSHandshakerUTLS(model.DiscardLogger, ClientHello),
				RootCAs:    srv.CertPool,
			}
			_, port, _ := net.SplitHostPort(srv.Addr())
			req, err := http.NewRequestWithContext(ctx, "GET", "https://www.example.com:"+port+"/", nil)
			if err != nil {
				t.Fatal(err)
			}
			req.Header.Set("Referer", "https://www.example.org/") // which Chrome wouldn't send
			body := fetch(t, txp, req)

			fp, err := srv.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			browserhttptest.Check(t, proto, fp, body)
		})
	}
}

func fetch(t *testing.T, txp model.HTTPTransport, req *http.Request) string {
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func TestDecodeBody(t *testing.T) {
	const page = browserhttptest.Body
	gz := func() []byte {
		var buf bytes.Buffer
		w := gzip.NewWriter(&buf)
		w.Write([]byte(page))
		w.Close()
		return buf.Bytes()
	}
	zl := func() []byte {
		var buf bytes.Buffer
		w := zlib.NewWriter(&buf)
		w.Write([]byte(page))
		w.Close()
		return buf.Bytes()
	}
	cases := []struct {
		encoding string
		body     []byte
		want     string
	}{
		{"gzip", gz(), page},
		{"deflate", zl(), page},
		{"br", browserhttptest.MustBrotli(page), page},
		{"zstd", browserhttptest.MustZstd(page), page},
		{"", []byte(page), page},
		{"gzip", nil, ""}, // e.g. a redirect: no body to decode
	}
	for _, tc := range cases {
		t.Run(tc.encoding, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{"Content-Encoding": {tc.encoding}},
				Body:   io.NopCloser(bytes.NewReader(tc.body)),
			}
			DecodeBody(resp)
			got, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if string(got) != tc.want {
				t.Errorf("got %q", got)
			}
		})
	}
}
