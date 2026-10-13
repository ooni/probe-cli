package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// testdata/sclient-*.txt is s_client's output against local s_server instances, recorded with
// OpenSSL 3.5.7 (sclient-VERSION.txt).
func TestSessionResultParsing(t *testing.T) {
	for _, c := range []struct {
		file, cipher, group string
		hrr                 bool
		tickets             int
	}{
		{"hrr-msg", "TLS_AES_256_GCM_SHA384", "ECDH, prime256v1, 256 bits", true, 0}, // server allows only P-256
		{"x25519", "TLS_AES_256_GCM_SHA384", "X25519, 253 bits", false, 0},
		{"mlkem", "TLS_AES_256_GCM_SHA384", "X25519MLKEM768", false, 0},
		{"tls12", "ECDHE-RSA-AES256-GCM-SHA384", "X25519, 253 bits", false, 0},
		{"fail-group", "", "", false, 0},
		{"tls1-default", "", "", false, 0}, // OpenSSL's default security level refuses TLS 1.0
		{"tls1-seclevel0", "ECDHE-RSA-AES256-SHA", "X25519, 253 bits", false, 0},
		{"tickets-eof", "TLS_AES_256_GCM_SHA384", "X25519MLKEM768", false, 0}, // stdin closed at once
		{"tickets-wait", "TLS_AES_256_GCM_SHA384", "X25519MLKEM768", false, 2},
	} {
		data, err := os.ReadFile(filepath.Join("testdata", "sclient-"+c.file+".txt"))
		if err != nil {
			t.Fatal(err)
		}
		r := sessionResult{string(data)}
		tickets := strings.Count(r.out, "Post-Handshake New Session Ticket arrived")
		if r.cipher() != c.cipher || r.ok() != (c.cipher != "") || r.group() != c.group ||
			r.helloRetryRequest() != c.hrr || tickets != c.tickets {
			t.Errorf("%s: cipher %q ok %v group %q hrr %v tickets %d", c.file, r.cipher(), r.ok(), r.group(),
				r.helloRetryRequest(), tickets)
		}
	}
}

// The leaves' extensions as production's are (openssl s_client -showcerts, 2026-10-13).
var productionLeafExtensions = []string{"extKeyUsage", "basicConstraints (critical)", "subjectKeyIdentifier",
	"authorityKeyIdentifier", "subjectAltName (critical)"}

func TestCertsAndChainDescription(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "certs")
	if err := writeCerts(dir); err != nil {
		t.Fatal(err)
	}
	read := func(name string) string {
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		return string(data)
	}
	for _, c := range []struct{ file, key, signature, issuer, dns string }{
		{"chat-ed25519.pem", "Ed25519", "SHA256-RSA", "CN=signal_client test CA (RSA),O=OONI signal_client test", "chat.signal.org"},
		{"chat-rsa.pem", "RSA 2048", "SHA256-RSA", "CN=signal_client test CA (RSA),O=OONI signal_client test", "chat.signal.org"},
		{"grpc-ed25519.pem", "Ed25519", "Ed25519", "CN=signal_client test CA (Ed25519),O=OONI signal_client test", "grpc.chat.signal.org"},
	} {
		chain := certificateChain(read(c.file))
		if len(chain) != 1 {
			t.Fatalf("%s: %d certificates", c.file, len(chain))
		}
		got := chain[0]
		if got.Subject != "" || got.Key != c.key || got.Signature != c.signature || got.Issuer != c.issuer ||
			!slices.Equal(got.DNSNames, []string{c.dns}) || !slices.Equal(got.Extensions, productionLeafExtensions) {
			t.Errorf("%s: %+v", c.file, got)
		}
	}
	cas := certificateChain(read("ca.pem"))
	if len(cas) != 2 || cas[0].Key != "RSA 4096" || cas[1].Key != "Ed25519" {
		t.Errorf("ca.pem: %+v", cas)
	}
}

func TestConfig(t *testing.T) {
	dir := t.TempDir()
	src, dst := filepath.Join(dir, "test.yml"), filepath.Join(dir, "out.yml")
	original := "server:\n  applicationConnectors:\n    - type: h2c\n      port: 8080\n" +
		"tlsKeyStore:\n  password: x\n" + "grpc:\n  port: 50051\n  websocketPort: 8080\n  h2c: true\n"
	if err := os.WriteFile(src, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeTestConfig(src, dst, "it's: #secret"); err != nil {
		t.Fatal(err)
	}
	out, err := os.ReadFile(dst)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"    - type: h2c\n      port: 8080\n    - type: h2\n      port: 8443\n",
		"      keyStorePassword: 'it''s: #secret'\n", "tlsKeyStore:\n  path: /certs/grpc.p12\n  password: x\n",
		"grpc:\n  bindAddress: 0.0.0.0\n  port: 8444\n  websocketPort: 8080\n  h2c: false\n"} {
		if !strings.Contains(string(out), want) {
			t.Errorf("no %q in\n%s", want, out)
		}
	}
	if err := os.WriteFile(src, []byte("server: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeTestConfig(src, dst, "x"); err == nil {
		t.Error("a changed test.yml was accepted")
	}
}

func TestH2Settings(t *testing.T) {
	srv := httptest.NewUnstartedServer(http.NotFoundHandler())
	srv.EnableHTTP2 = true
	srv.StartTLS()
	defer srv.Close()
	res, err := h2Settings("example.org", srv.Listener.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	// Go's server sends MAX_CONCURRENT_STREAMS 250 and a connection WINDOW_UPDATE
	if res.ALPN != "h2" || res.Settings["MAX_CONCURRENT_STREAMS"] != 250 || res.WindowUpdate == nil {
		t.Errorf("%+v", res)
	}
}

func TestParseCipherList(t *testing.T) {
	out := "TLS_AES_256_GCM_SHA384         - TLS_AES_256_GCM_SHA384 TLSv1.3 Kx=any Au=any Enc=AESGCM(256) Mac=AEAD\n" +
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256 - ECDHE-RSA-AES128-GCM-SHA256 TLSv1.2 Kx=ECDH Au=RSA Enc=AESGCM(128) Mac=AEAD\n" +
		"TLS_RSA_WITH_AES_128_CBC_SHA          - AES128-SHA                     SSLv3 Kx=RSA Au=RSA Enc=AES(128) Mac=SHA1\n"
	want := []cipherSuite{{"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256", "ECDHE-RSA-AES128-GCM-SHA256"},
		{"TLS_RSA_WITH_AES_128_CBC_SHA", "AES128-SHA"}}
	if got := parseCipherList(out); !slices.Equal(got, want) {
		t.Errorf("%q", got)
	}
}
