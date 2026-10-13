package signalclient

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	utls "github.com/refraction-networking/utls"
)

// captureSNI is the server name each hello was captured with.
var captureSNI = map[string]string{
	HelloLibsignal.Name:        "grpc.chat.signal.org",
	HelloLibsignalEnclave.Name: "grpc.chat.signal.org",
	HelloAndroidWebSocket.Name: "chat.signal.org",
	HelloAndroidREST.Name:      "cdn2.signal.org",
	HelloDesktop.Name:          "updates2.signal.org",

	HelloLibsignalReflector.Name:     "github.githubassets.com",
	HelloAndroidGmailWebSocket.Name:  "www.google.com",
	HelloAndroidGmailREST.Name:       "www.google.com",
	HelloAndroidGmapsWebSocket.Name:  "clients3.google.com",
	HelloAndroidGmapsREST.Name:       "clients3.google.com",
	HelloAndroidModernWebSocket.Name: "github.githubassets.com",
	HelloAndroidModernREST.Name:      "github.githubassets.com",
}

// capturedHellos are all the hellos captured from Signal's clients.
var capturedHellos = []Hello{HelloLibsignal, HelloLibsignalEnclave, HelloLibsignalReflector, HelloAndroidWebSocket,
	HelloAndroidREST, HelloAndroidGmailWebSocket, HelloAndroidGmailREST, HelloAndroidGmapsWebSocket, HelloAndroidGmapsREST,
	HelloAndroidModernWebSocket, HelloAndroidModernREST, HelloDesktop}

// generatedHello builds the ClientHello uTLS would send for h, with the SNI it was captured with.
func generatedHello(t *testing.T, h Hello) []byte {
	t.Helper()
	return generatedHelloFor(t, h, captureSNI[h.Name])
}

// generatedHelloFor builds the ClientHello uTLS would send for h to sni.
func generatedHelloFor(t *testing.T, h Hello, sni string) []byte {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	uconn := utls.UClient(client, &utls.Config{ServerName: sni}, utls.HelloCustom)
	if err := h.apply(uconn); err != nil {
		t.Fatal(err)
	}
	if err := uconn.BuildHandshakeState(); err != nil {
		t.Fatal(err)
	}
	return uconn.HandshakeState.Hello.Raw
}

// helloShape is what must be identical between a captured and a generated hello: everything
// except the random, session id and key share values.
type helloShape struct {
	size       int
	sessionID  int // the legacy session id's length
	ciphers    []uint16
	extensions []uint16
	groups     []utls.CurveID
	sigalgs    []utls.SignatureScheme
	alpn       []string
}

func shapeOf(t *testing.T, raw []byte) helloShape {
	t.Helper()
	record := append([]byte{0x16, 0x03, 0x01, byte(len(raw) >> 8), byte(len(raw))}, raw...)
	spec, err := (&utls.Fingerprinter{}).FingerprintClientHello(record)
	if err != nil {
		t.Fatal(err)
	}
	s := helloShape{size: len(raw), sessionID: int(raw[38]), ciphers: spec.CipherSuites, extensions: extensionOrder(t, raw)}
	for _, ext := range spec.Extensions {
		switch e := ext.(type) {
		case *utls.SupportedCurvesExtension:
			s.groups = e.Curves
		case *utls.SignatureAlgorithmsExtension:
			s.sigalgs = e.SupportedSignatureAlgorithms
		case *utls.ALPNExtension:
			s.alpn = e.AlpnProtocols
		}
	}
	return s
}

// extensionOrder lists the extension types of a ClientHello handshake message, in order.
func extensionOrder(t *testing.T, b []byte) (out []uint16) {
	t.Helper()
	i := 4 + 2 + 32
	i += 1 + int(b[i])                          // session id
	i += 2 + (int(b[i])<<8 | int(b[i+1]))       // cipher suites
	i += 1 + int(b[i])                          // compression methods
	end := i + 2 + (int(b[i])<<8 | int(b[i+1])) // extensions block
	for i += 2; i+4 <= end; {
		out = append(out, uint16(b[i])<<8|uint16(b[i+1]))
		i += 4 + (int(b[i+2])<<8 | int(b[i+3]))
	}
	return out
}

func TestHellosReproduceTheCapturedOnes(t *testing.T) {
	for _, h := range capturedHellos {
		if h.alpn != nil {
			continue // derived from a capture (TestEnclaveHelloOnlyChangesALPN)
		}
		t.Run(h.Name, func(t *testing.T) {
			want, got := shapeOf(t, h.raw), shapeOf(t, generatedHello(t, h))
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("generated hello differs from the capture:\nwant %+v\ngot  %+v", want, got)
			}
		})
	}
}

func TestEnclaveHelloOnlyChangesALPN(t *testing.T) {
	base, enclave := shapeOf(t, HelloLibsignal.raw), shapeOf(t, generatedHello(t, HelloLibsignalEnclave))
	if !reflect.DeepEqual(enclave.alpn, []string{"http/1.1"}) {
		t.Fatalf("enclave ALPN: %v", enclave.alpn)
	}
	base.alpn, enclave.alpn = nil, nil
	base.size, enclave.size = 0, 0 // "h2" vs "http/1.1" changes the length
	if !reflect.DeepEqual(base, enclave) {
		t.Fatalf("enclave hello differs beyond ALPN:\nbase    %+v\nenclave %+v", base, enclave)
	}
}

func TestCapturedHelloSizes(t *testing.T) {
	// the sizes are part of the fingerprint: the Android hellos with an ML-KEM key share span two
	// TCP segments
	for _, h := range capturedHellos {
		if h.alpn != nil {
			continue
		}
		if got := len(generatedHello(t, h)); got != len(h.raw) {
			t.Errorf("%s: generated %d bytes, captured %d", h.Name, got, len(h.raw))
		}
	}
	// libsignal pads its reflector hello to 512 bytes whatever the front's name (captured on all
	// eight fronts)
	for _, sni := range append(append([]string{}, fastlyFronts...), "www.google.com", "android.clients.google.com",
		"clients3.google.com", "clients4.google.com", "googlemail.com") {
		if got := len(generatedHelloFor(t, HelloLibsignalReflector, sni)); got != 512 {
			t.Errorf("reflector hello to %s: %d bytes", sni, got)
		}
	}
}

// Each front gets the hello Signal's client sends to it.
func TestFrontHello(t *testing.T) {
	for _, c := range []struct {
		route, service, front string
		want                  Hello
	}{
		{RouteReflectorF, "", "pinterest.com", HelloLibsignalReflector},
		{RouteReflectorG, "", "clients3.google.com", HelloLibsignalReflector},
		{RouteAndroidF, "chat", "www.redditstatic.com", HelloAndroidModernWebSocket},
		{RouteAndroidF, "storage", "pinterest.com", HelloAndroidModernREST},
		{RouteAndroidG, "chat", "www.google.com.eg", HelloAndroidGmailWebSocket},
		{RouteAndroidG, "storage", "android.clients.google.com", HelloAndroidGmailREST},
		{RouteAndroidG, "chat", "clients4.google.com", HelloAndroidGmapsWebSocket},
		{RouteAndroidG, "storage", "clients3.google.com", HelloAndroidGmapsREST},
	} {
		if got := frontHello(c.route, c.service, c.front); got.Name != c.want.Name {
			t.Errorf("%s %s %s: %s, want %s", c.route, c.service, c.front, got.Name, c.want.Name)
		}
	}
}

// sentHello runs a real handshake with h and returns the ClientHello bytes actually written to
// the connection (the first handshake message), not uTLS's pre-built copy.
func sentHello(t *testing.T, h Hello) []byte {
	t.Helper()
	client, server := net.Pipe()
	defer client.Close()
	presetErr := make(chan error, 1)
	go func() {
		uconn := utls.UClient(client, &utls.Config{ServerName: captureSNI[h.Name]}, utls.HelloCustom)
		if err := h.apply(uconn); err != nil {
			presetErr <- err
			client.Close() // so the reads below fail instead of waiting forever
			return
		}
		_ = uconn.Handshake() // fails when the server side closes; we only want its first flight
	}()
	readFull := func(b []byte) {
		if _, err := io.ReadFull(server, b); err != nil {
			select {
			case perr := <-presetErr:
				t.Fatalf("ApplyPreset: %v", perr)
			default:
				t.Fatal(err)
			}
		}
	}
	// read TLS records until the handshake message is complete (a large hello may span records)
	var hs []byte
	msgLen := func() int { return 4 + (int(hs[1])<<16 | int(hs[2])<<8 | int(hs[3])) }
	for len(hs) < 4 || len(hs) < msgLen() {
		var hdr [5]byte
		readFull(hdr[:])
		body := make([]byte, int(hdr[3])<<8|int(hdr[4]))
		readFull(body)
		hs = append(hs, body...)
	}
	server.Close()
	return hs[:msgLen()]
}

func TestHandshakeSendsTheCapturedHellos(t *testing.T) {
	for _, h := range capturedHellos {
		t.Run(h.Name, func(t *testing.T) {
			want, got := shapeOf(t, h.raw), shapeOf(t, sentHello(t, h))
			if h.alpn != nil { // Spec rewrites the capture's ALPN (checked against it above): compare with the build
				want = shapeOf(t, generatedHello(t, h))
			}
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("the hello sent on the wire differs from the capture:\nwant %+v\ngot  %+v", want, got)
			}
		})
	}
}

// Signal-Android's Google-front hellos offer only TLS 1.2; the controls' hellos negotiate
// what their names say.
func TestHelloVersions(t *testing.T) {
	for _, h := range []Hello{HelloAndroidGmailWebSocket, HelloAndroidGmailREST, HelloAndroidGmapsWebSocket, HelloAndroidGmapsREST} {
		if v := handshakeVersion(t, h); v != tls.VersionTLS12 {
			t.Errorf("%s negotiated %#x", h.Name, v)
		}
	}
	for _, h := range []Hello{HelloLibsignalReflector, HelloAndroidModernWebSocket, HelloAndroidModernREST} {
		if v := handshakeVersion(t, h); v != tls.VersionTLS13 {
			t.Errorf("%s negotiated %#x", h.Name, v)
		}
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
	certPEM, keyPEM := mustSelfSigned(t, "example.com")
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
	tr, err := handshakeTLS(ctx, measurexlite.NewTrace(0, time.Now()), client, "pipe", "example.com", h, nil)
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
