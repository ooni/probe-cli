package main

import (
	"archive/zip"
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
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
	for _, changed := range []string{"server: {}\n", original + original} { // an excerpt missing, or twice
		if err := os.WriteFile(src, []byte(changed), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := writeTestConfig(src, dst, "x"); err == nil {
			t.Errorf("accepted %q", changed)
		}
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

func TestTargets(t *testing.T) {
	for target, want := range map[string]string{"chat.signal.org": "chat.signal.org:443",
		"chat.signal.org@127.0.0.1:8443": "127.0.0.1:8443", "chat.signal.org@[::1]:8443": "[::1]:8443",
		"x@127.0.0.1": "", "[::1]": "", "@127.0.0.1:1": ""} {
		_, addr, err := splitTarget(target)
		if addr != want || (err == nil) != (want != "") {
			t.Errorf("%q: %q, %v", target, addr, err)
		}
	}
}

func TestUnreachableTargetFailsFast(t *testing.T) {
	if _, err := exec.LookPath("openssl"); err != nil {
		t.Skip("no openssl")
	}
	start := time.Now()
	var out bytes.Buffer
	err := tlsProfiles(&out, []string{"x@127.0.0.1:1"})
	if err == nil || !strings.Contains(out.String(), `"error": "could not connect to 127.0.0.1:1"`) ||
		strings.Contains(out.String(), "versions") || time.Since(start) > 5*time.Second {
		t.Errorf("err %v after %v:\n%s", err, time.Since(start), out.String())
	}
}

func TestUnpackMaven(t *testing.T) {
	dir := t.TempDir()
	zipWith := func(names ...string) string {
		path := filepath.Join(dir, fmt.Sprintf("m%d.zip", len(names)))
		f, err := os.Create(path)
		if err != nil {
			t.Fatal(err)
		}
		w := zip.NewWriter(f)
		for _, name := range names {
			if _, err := w.Create(name); err != nil {
				t.Fatal(err)
			}
		}
		if err := w.Close(); err != nil {
			t.Fatal(err)
		}
		return path
	}
	good := zipWith("apache-maven/", "apache-maven/bin/mvn", "apache-maven/lib/x.jar")
	if err := unpackMaven(good, filepath.Join(dir, "maven")); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(dir, "maven", "bin", "mvn")); err != nil || info.Mode().Perm()&0o100 == 0 {
		t.Errorf("bin/mvn: %v, %v", info, err)
	}
	if err := unpackMaven(good, filepath.Join(dir, "maven")); err == nil {
		t.Error("unpacked over an existing directory")
	}
	if err := unpackMaven(zipWith("apache-maven/bin/mvn", "other/x"), filepath.Join(dir, "m2")); err == nil {
		t.Error("accepted an entry outside the top directory")
	}
	if _, err := os.Stat(filepath.Join(dir, "m2.tmp")); err == nil {
		t.Error("left m2.tmp behind")
	}
}
