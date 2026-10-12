package signalclient

import (
	"crypto/x509"
	"io"
	"net"
	"reflect"
	"testing"

	utls "github.com/refraction-networking/utls"
)

// captureSNI is the server name each hello was captured with.
var captureSNI = map[string]string{
	HelloLibsignal.Name:        "grpc.chat.signal.org",
	HelloLibsignalEnclave.Name: "grpc.chat.signal.org",
	HelloAndroidWebSocket.Name: "chat.signal.org",
	HelloAndroidREST.Name:      "cdn2.signal.org",
	HelloDesktop.Name:          "updates2.signal.org",
}

// generatedHello builds the ClientHello uTLS would send for h.
func generatedHello(t *testing.T, h Hello) []byte {
	t.Helper()
	spec, err := h.Spec()
	if err != nil {
		t.Fatal(err)
	}
	client, server := net.Pipe()
	defer client.Close()
	defer server.Close()
	uconn := utls.UClient(client, &utls.Config{ServerName: captureSNI[h.Name]}, utls.HelloCustom)
	if err := uconn.ApplyPreset(spec); err != nil {
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
	s := helloShape{size: len(raw), ciphers: spec.CipherSuites, extensions: extensionOrder(t, raw)}
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
	for _, h := range []Hello{HelloLibsignal, HelloAndroidWebSocket, HelloAndroidREST, HelloDesktop} {
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
	// the sizes are part of the fingerprint: the Android hello spans two TCP segments
	for h, size := range map[*Hello]int{&HelloLibsignal: 212, &HelloAndroidWebSocket: 1473, &HelloAndroidREST: 1476, &HelloDesktop: 1475} {
		if got := len(generatedHello(t, *h)); got != size {
			t.Errorf("%s: generated %d bytes, captured %d", h.Name, got, size)
		}
	}
}

func TestSignalRoots(t *testing.T) {
	pool, err := signalRoots()
	if err != nil {
		t.Fatal(err)
	}
	for _, der := range [][]byte{signalRSARootDER, signalEd25519RootDER} {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
			t.Fatalf("%s does not verify against the pool: %v", cert.Subject, err)
		}
	}
}

// sentHello runs a real handshake with h and returns the ClientHello bytes actually written to
// the connection (the first handshake message), not uTLS's pre-built copy.
func sentHello(t *testing.T, h Hello) []byte {
	t.Helper()
	spec, err := h.Spec()
	if err != nil {
		t.Fatal(err)
	}
	client, server := net.Pipe()
	defer client.Close()
	presetErr := make(chan error, 1)
	go func() {
		uconn := utls.UClient(client, &utls.Config{ServerName: captureSNI[h.Name]}, utls.HelloCustom)
		if err := uconn.ApplyPreset(spec); err != nil {
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
	for _, h := range []Hello{HelloLibsignal, HelloLibsignalEnclave, HelloAndroidWebSocket, HelloAndroidREST, HelloDesktop} {
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
