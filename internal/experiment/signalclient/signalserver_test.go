//go:build signalserver

package signalclient

//
// Opt-in checks against Signal-Server's own code (its feature-limited test mode, started by
// signalserver/run.sh). Built only with -tags signalserver: never part of the normal test run,
// and probe-cli does not depend on any of it. Run with signalserver/test.sh.
//

import (
	"context"
	"crypto/x509"
	"os"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// signalServer returns the address of the local connector shaped like sni's production front end
// and a pool with the local test CAs.
func signalServer(t *testing.T, sni string) (string, *x509.CertPool) {
	env := "SIGNAL_SERVER_ADDR"
	if sni == "grpc.chat.signal.org" {
		env = "SIGNAL_SERVER_GRPC_ADDR"
	}
	addr, caFile := os.Getenv(env), os.Getenv("SIGNAL_SERVER_CA")
	if addr == "" || caFile == "" {
		t.Skip(env + " and SIGNAL_SERVER_CA not set: start signalserver/run.sh and use signalserver/test.sh")
	}
	pem, err := os.ReadFile(caFile)
	if err != nil {
		t.Fatal(err)
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(pem) {
		t.Fatal("no certificate in SIGNAL_SERVER_CA")
	}
	return addr, pool
}

// provisionAgainstSignalServer runs one route of the experiment's flow against the local server.
func provisionAgainstSignalServer(t *testing.T, hello Hello, sni string, h2 bool) {
	addr, roots := signalServer(t, sni)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	zero := time.Now()
	trace := measurexlite.NewTrace(0, zero)
	tr, err := dialTLS(ctx, trace, model.DiscardLogger, addr, sni, hello, roots)
	if err != nil {
		t.Fatalf("TLS with %s: %v", hello.Name, err)
	}
	defer tr.conn.Close()
	t.Logf("TLS ok: ALPN %q, leaf %s", tr.alpn, tr.leafKey)
	var up *upgradeResult
	if h2 {
		up, err = websocketH2(ctx, ctx, tr.conn, sni, "/v1/websocket/provisioning/", ooniUserAgent+" libsignal/0.105.0")
	} else {
		up, err = websocketH1(ctx, tr.conn, sni, "/v1/websocket/provisioning/", ooniUserAgent)
	}
	if err != nil {
		status := 0
		if up != nil {
			status = up.status
		}
		t.Fatalf("upgrade: status %d: %v", status, err)
	}
	defer up.closer.Close()
	t.Logf("upgrade: status %d, x-signal-timestamp %v, extensions %q", up.status, up.signalTimestamp, up.extensions)
	res := runProvisioning(ctx, up.ws, zero, newMeasurer(Config{}).provisioningFlow(), nil)
	if res.Failure != nil {
		t.Fatalf("provisioning failed at %s: %s (%+v)", res.FailedStep, *res.Failure, res)
	}
	if !res.AddressReceived || !res.Acked || res.PingEchoed == nil || !*res.PingEchoed {
		t.Fatalf("incomplete flow: %+v", res)
	}
	for _, k := range res.Keepalives {
		t.Logf("keepalive padding %5d: status %d id_match %v timestamp %v", k.Padding, k.Status, k.IDMatch, k.SignalTimestamp)
		if k.Status != 200 || !k.IDMatch {
			t.Errorf("keepalive %+v", k)
		}
	}
	t.Logf("address of %d chars received %.3f s after start", res.AddressLength, res.AddressT)
}

// Android's route: HTTP/1.1 upgrade, offering permessage-deflate as OkHttp does.
func TestSignalServerProvisioningH1(t *testing.T) {
	provisionAgainstSignalServer(t, HelloAndroidWebSocket, "chat.signal.org", false)
}

// libsignal's route: RFC 8441 extended CONNECT over HTTP/2. Production serves it from
// grpc.chat.signal.org; whether Signal-Server's own Jetty connector does is what this checks.
func TestSignalServerProvisioningH2(t *testing.T) {
	provisionAgainstSignalServer(t, HelloLibsignal, "grpc.chat.signal.org", true)
}
