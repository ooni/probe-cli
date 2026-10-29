package webconnectivitylte

import (
	"context"
	"crypto/tls"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/browserhttp"
	"github.com/ooni/probe-cli/v3/internal/browserhttp/browserhttptest"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// SecureFlow's handshake and request present Chrome's fingerprint.
func TestSecureFlowFingerprint(t *testing.T) {
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

			// the same handshake and request as SecureFlow.Run
			trace := measurexlite.NewTrace(0, time.Now())
			thx := trace.NewTLSHandshakerUTLS(model.DiscardLogger, browserhttp.ClientHello)
			sf := &SecureFlow{Address: "93.184.215.14:443", HostHeader: "www.example.com", URLPath: "/"}
			tlsConn, err := thx.Handshake(ctx, conn, &tls.Config{
				NextProtos: sf.alpn(),
				RootCAs:    srv.CertPool,
				ServerName: "www.example.com",
			})
			if err != nil {
				t.Fatal(err)
			}
			alpn := netxlite.MaybeTLSConnectionState(tlsConn).NegotiatedProtocol
			req, err := sf.newHTTPRequest(ctx)
			if err != nil {
				t.Fatal(err)
			}
			resp, err := browserhttp.NewTransport(tlsConn, alpn).RoundTrip(req)
			if err != nil {
				t.Fatal(err)
			}
			body, err := netxlite.ReadAllContext(ctx, resp.Body)
			resp.Body.Close()
			if err != nil {
				t.Fatal(err)
			}

			fp, err := srv.Fingerprint()
			if err != nil {
				t.Fatal(err)
			}
			browserhttptest.Check(t, proto, fp, string(body))
		})
	}
}
