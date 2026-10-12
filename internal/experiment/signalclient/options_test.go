package signalclient

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	utls "github.com/refraction-networking/utls"
)

func TestParseControls(t *testing.T) {
	for in, want := range map[string]string{"": "on_failure", "on_failure": "on_failure", "always": "always", "off": "off"} {
		if got, err := parseControls(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	if _, err := parseControls("sometimes"); err == nil {
		t.Error("accepted an invalid value")
	}
}

func TestParseCircumvention(t *testing.T) {
	for in, want := range map[string]string{
		"":                        "",
		"all":                     "reflector_f,reflector_g,android_g,android_f",
		"android_f, reflector_g":  "reflector_g,android_f",
		"reflector_f,reflector_f": "reflector_f",
	} {
		got, err := parseCircumvention(in)
		if err != nil || strings.Join(got, ",") != want {
			t.Errorf("%q: %v %v", in, got, err)
		}
	}
	if _, err := parseCircumvention("reflector_x"); err == nil {
		t.Error("accepted an unknown route")
	}
}

func TestParseStaticAddresses(t *testing.T) {
	got, err := parseStaticAddresses("chat.signal.org=13.248.212.111, 2600::1 ; CDN.signal.org=1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if len(got["chat.signal.org"]) != 2 || got["chat.signal.org"][1].ip != "2600::1" || got["cdn.signal.org"][0].source != "option" {
		t.Errorf("%+v", got)
	}
	for _, bad := range []string{"chat.signal.org", "chat.signal.org=not-an-ip", "=1.2.3.4"} {
		if _, err := parseStaticAddresses(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestBuiltinStaticAddresses(t *testing.T) {
	all := builtinStaticAddresses()
	v4 := 0
	for _, list := range all {
		for _, a := range list {
			if net.ParseIP(a.ip) == nil {
				t.Errorf("not an IP: %q", a.ip)
			}
			if net.ParseIP(a.ip).To4() != nil {
				v4++
			}
		}
	}
	if v4 != 36 { // the spec's 34 distinct IPv4 addresses, with cdn2 and cdn3 sharing 2
		t.Errorf("%d IPv4 static addresses", v4)
	}
	cdsi := all["cdsi.signal.org"]
	if len(cdsi) != 2 || cdsi[0].source != "libsignal" || cdsi[1].source != "libsignal" {
		t.Errorf("cdsi: %+v (an address keeps the first list that has it)", cdsi)
	}
	got := staticCandidates(all["svr2.signal.org"], "20.236.21.158")
	if len(got) != 3 || got[0].ip != "20.104.52.125" || got[1].ip != "20.9.45.98" || net.ParseIP(got[2].ip).To4() != nil {
		t.Errorf("candidates %+v", got)
	}
	if first, ok := firstStatic(all["updates2.signal.org"]); !ok || first.ip != "104.18.26.44" || first.source != "desktop" {
		t.Errorf("first static %+v", first)
	}
	if _, ok := firstStatic(nil); ok {
		t.Error("first static of an empty list")
	}
}

func TestFronts(t *testing.T) {
	f := builtinFronts("EG")
	if f["android_g.chat"].snis[0] != "www.google.com.eg" || len(f["android_g.chat"].snis) != 6 ||
		f["android_g.storage"].pathPrefix != "/storage" || f["reflector_g"].snis[0] != "www.google.com" {
		t.Errorf("EG: %+v", f)
	}
	if f := builtinFronts("BR"); f["android_g.chat"].snis[0] != "www.google.com" {
		t.Errorf("BR: %+v", f["android_g.chat"])
	}
	if err := parseFronts("reflector_f=r.example@a.example,b.example;android_g=h.example@c.example", f); err != nil {
		t.Fatal(err)
	}
	if fc := f["reflector_f"]; fc.hostHeader != "r.example" || strings.Join(fc.snis, ",") != "a.example,b.example" {
		t.Errorf("reflector_f %+v", fc)
	}
	if fc := f["android_g.storage"]; fc.hostHeader != "h.example" || fc.snis[0] != "c.example" || fc.pathPrefix != "/storage" {
		t.Errorf("android_g.storage %+v (the path prefix is kept)", fc)
	}
	for _, bad := range []string{"reflector_f=host", "reflector_x=h@s", "reflector_f.chat=h@s", "android_f=h@", "=h@s"} {
		if err := parseFronts(bad, builtinFronts("")); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestPickFront(t *testing.T) {
	if got := pickFront(RouteAndroidG, "EG", builtinFronts("EG")["android_g.chat"].snis); got != "www.google.com.eg" {
		t.Errorf("android_g EG: %s", got)
	}
	if got := pickFront(RouteAndroidF, "BR", fastlyFronts); got != fastlyFronts[0] {
		t.Errorf("android_f BR: %s", got)
	}
	seen := map[string]bool{}
	for range 200 {
		seen[pickFront(RouteReflectorF, "", fastlyFronts)] = true
		seen[pickFront(RouteAndroidF, "IR", fastlyFronts)] = true
	}
	if len(seen) != len(fastlyFronts) {
		t.Errorf("random fronts never picked some: %v", seen)
	}
}

func TestDecoySNI(t *testing.T) {
	for _, host := range []string{"grpc.chat.signal.org", "cdn.signal.org", "a.b"} {
		d := decoySNI(len(host))
		if !strings.HasSuffix(d, ".example.com") || (len(host) > len(".example.com") && len(d) != len(host)) {
			t.Errorf("%s: %s", host, d)
		}
		if strings.ToLower(d) != d {
			t.Errorf("not lowercase: %s", d)
		}
	}
}

func TestParsePEMBundle(t *testing.T) {
	if _, err := parsePEMBundle("not pem"); err == nil {
		t.Error("accepted a bundle without certificates")
	}
	if _, err := parsePEMBundle("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"); err == nil {
		t.Error("accepted a key")
	}
	certs, err := parsePEMBundle(mustSelfSignedPEM(t) + mustSelfSignedPEM(t))
	if err != nil || len(certs) != 2 {
		t.Errorf("%d %v", len(certs), err)
	}
	if len(newTrustAnchors(certs).sha256[0]) != 64 {
		t.Error("not a hex SHA-256")
	}
}

func TestGoogleRoots(t *testing.T) {
	pool, err := googleRoots()
	if err != nil {
		t.Fatal(err)
	}
	//lint:ignore SA1019 counting the embedded roots
	if n := len(pool.Subjects()); n != 7 {
		t.Errorf("%d Google roots", n)
	}
}

func TestOptionsAreValidatedBeforeMeasuring(t *testing.T) {
	for _, config := range []Config{{Controls: "x"}, {Circumvention: "x"}, {Fronts: "x"}, {StaticAddresses: "x"}, {SignalCA: "x"}} {
		m := NewExperimentMeasurer(config).(*Measurer)
		if _, err := m.parseOptions("IT"); err == nil {
			if _, err := m.signalAnchors(); err == nil {
				t.Errorf("%+v accepted", config)
			}
		}
	}
	m := NewExperimentMeasurer(Config{CircumventionCountry: " pk "}).(*Measurer)
	if opts, err := m.parseOptions("IT"); err != nil || opts.country != "PK" {
		t.Errorf("%+v %v", opts, err)
	}
	m = NewExperimentMeasurer(Config{}).(*Measurer)
	if opts, err := m.parseOptions("ir"); err != nil || opts.country != "IR" || opts.controls != ControlsOnFailure {
		t.Errorf("%+v %v", opts, err)
	}
}

// A hello limited to TLS 1.2 drops the TLS 1.3 extensions and negotiates TLS 1.2.
func TestHelloLimitedToTLS12(t *testing.T) {
	h := HelloAndroidWebSocket.limitedToTLS12()
	if h.Name != HelloAndroidWebSocket.Name+"-tls12" {
		t.Errorf("name %s", h.Name)
	}
	spec, err := h.Spec()
	if err != nil {
		t.Fatal(err)
	}
	for _, ext := range spec.Extensions {
		switch ext.(type) {
		case *utls.SupportedVersionsExtension, *utls.KeyShareExtension, *utls.PSKKeyExchangeModesExtension:
			t.Errorf("TLS 1.3 extension %T kept", ext)
		}
	}
	if v := handshakeVersion(t, h); v != tls.VersionTLS12 {
		t.Errorf("negotiated %#x", v)
	}
	if v := handshakeVersion(t, HelloChrome); v != tls.VersionTLS13 {
		t.Errorf("chrome negotiated %#x", v)
	}
	if v := handshakeVersion(t, helloStdlibTLS12(HelloLibsignal)); v != tls.VersionTLS12 {
		t.Errorf("stdlib negotiated %#x", v)
	}
	if got := helloStdlibTLS12(HelloLibsignal).ALPN(); strings.Join(got, ",") != "h2" {
		t.Errorf("stdlib ALPN %v", got)
	}
	if a, b := helloStdlibTLS12(HelloAndroidREST).Name, helloStdlibTLS12(HelloDesktop).Name; a != "go-crypto-tls-1.2+h2,http/1.1" ||
		b != "go-crypto-tls-1.2" {
		t.Errorf("stdlib names %q %q", a, b)
	}
	if !strings.HasPrefix(HelloChrome.Name, "utls-chrome-") {
		t.Errorf("chrome name %s", HelloChrome.Name)
	}
}

// handshakeVersion runs h against a local Go TLS server (unverified) and returns the version.
func handshakeVersion(t *testing.T, h Hello) uint16 {
	t.Helper()
	certPEM, keyPEM := mustSelfSignedPair(t)
	cert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		t.Fatal(err)
	}
	// a loopback socket, not net.Pipe: TLS 1.3 flights that both sides write at once would block
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()
	go func() {
		server, err := ln.Accept()
		if err != nil {
			return
		}
		defer server.Close()
		s := tls.Server(server, &tls.Config{Certificates: []tls.Certificate{cert}, NextProtos: []string{"h2", "http/1.1"}})
		_ = s.Handshake()
		_, _ = s.Read(make([]byte, 1)) // until the client closes
	}()
	client, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	tr, err := handshakeTLS(ctx, measurexlite.NewTrace(0, time.Now()), time.Now(), client, "pipe", "example.com", h, nil)
	if err != nil {
		t.Fatalf("%s: %v", h.Name, err)
	}
	defer tr.conn.Close()
	switch tr.handshake.TLSVersion {
	case "TLSv1.2":
		return tls.VersionTLS12
	case "TLSv1.3":
		return tls.VersionTLS13
	}
	t.Fatalf("version %s", tr.handshake.TLSVersion)
	return 0
}

// mustSelfSignedPair returns a self-signed ECDSA certificate and key for example.com.
func mustSelfSignedPair(t *testing.T) ([]byte, []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(2), Subject: pkix.Name{CommonName: "example.com"},
		DNSNames: []string{"example.com"}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour)}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kder, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder})
}

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "signal_client" || m.ExperimentVersion() != "0.2.0" {
		t.Errorf("%s %s", m.ExperimentName(), m.ExperimentVersion())
	}
}
