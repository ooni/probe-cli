package signalclient

//
// The TLS ClientHellos of real Signal clients, reproduced byte for byte with uTLS.
//
// Each file in hellos/ is a ClientHello handshake message (starting with type 1) captured from
// the client's own network code; see hellos/README.md for how each was captured. We send the
// same hello the real client sends so that censorship keyed on Signal's fingerprint is measured.
//

import (
	"crypto/x509"
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math"

	utls "github.com/refraction-networking/utls"
)

var (
	//go:embed hellos/libsignal.bin
	rawHelloLibsignal []byte

	//go:embed hellos/android-okhttp-websocket.bin
	rawHelloAndroidWebSocket []byte

	//go:embed hellos/android-okhttp-rest.bin
	rawHelloAndroidREST []byte

	//go:embed hellos/desktop-electron.bin
	rawHelloDesktop []byte

	//go:embed hellos/signal.cer
	signalRSARootDER []byte

	//go:embed hellos/signal-ed25519.cer
	signalEd25519RootDER []byte
)

// Hello is one Signal client handshake that this experiment reproduces.
type Hello struct {
	// Name identifies the hello in the measurement.
	Name string

	// raw is the captured ClientHello handshake message.
	raw []byte

	// alpn, when not nil, replaces the captured ALPN protocols. Only used where the client
	// sends the same hello with a different ALPN (libsignal's enclave connections).
	alpn []string

	// appUserAgent is the User-Agent the real client sends (used with Config.AppUserAgent).
	appUserAgent string

	// libsignal is true for libsignal's routes, whose User-Agent ends in " libsignal/<version>".
	libsignal bool
}

var (
	// HelloLibsignal is libsignal 0.105.0's direct-route hello, the version Signal-Android and
	// Signal-iOS pin (chat on both, provisioning on iOS): TLS 1.3 only, groups x25519, P-256 and
	// P-384 with an x25519 key share, Ed25519 first, ALPN h2. Signal-Desktop pinned libsignal
	// 0.103.0 when this was captured; its hello was not captured separately.
	HelloLibsignal = Hello{Name: "libsignal-0.105.0", raw: rawHelloLibsignal, libsignal: true,
		appUserAgent: "Signal-Android/8.29.3 Android/34 libsignal/0.105.0"}

	// HelloLibsignalEnclave is the same hello with ALPN http/1.1, which libsignal uses for the
	// CDSI, SVR2 and SVR-B enclaves (same ssl_config, alpn parameter differs; net/src/env.rs).
	HelloLibsignalEnclave = Hello{Name: "libsignal-0.105.0-enclave", raw: rawHelloLibsignal, alpn: []string{"http/1.1"},
		libsignal: true, appUserAgent: "Signal-Android/8.29.3 Android/34 libsignal/0.105.0"}

	// HelloAndroidWebSocket is Signal-Android 8.29.3's OkHttp 5.3.2 + Conscrypt 2.6.2 hello for
	// WebSockets (provisioning): ALPN http/1.1, X25519MLKEM768 key share (1473 bytes).
	HelloAndroidWebSocket = Hello{Name: "android-okhttp-5.3.2-conscrypt-2.6.2-websocket", raw: rawHelloAndroidWebSocket,
		appUserAgent: "Signal-Android/8.29.3 Android/34"}

	// HelloAndroidREST is the same client's hello for HTTPS (storage, CDNs): ALPN h2,http/1.1.
	HelloAndroidREST = Hello{Name: "android-okhttp-5.3.2-conscrypt-2.6.2", raw: rawHelloAndroidREST,
		appUserAgent: "Signal-Android/8.29.3 Android/34"}

	// HelloDesktop is Signal-Desktop's own fetches (updater, WebAPI's agent) through Electron
	// 44.4.5's Node and BoringSSL: no ALPN, X25519MLKEM768 + x25519 key shares (1475 bytes).
	HelloDesktop = Hello{Name: "desktop-electron-44.4.5", raw: rawHelloDesktop,
		appUserAgent: "Signal-Desktop/8.33.0 Linux"}
)

// errNoALPN means a hello has no ALPN extension to replace.
var errNoALPN = errors.New("signal_client: captured hello has no ALPN extension")

// Spec returns a uTLS spec that reproduces the captured hello.
func (h Hello) Spec() (*utls.ClientHelloSpec, error) {
	if len(h.raw) < 4 || h.raw[0] != 0x01 || len(h.raw) > math.MaxUint16 {
		return nil, fmt.Errorf("signal_client: %s: not a ClientHello handshake message", h.Name)
	}
	// the fingerprinter expects a TLS record around the handshake message
	record := binary.BigEndian.AppendUint16([]byte{0x16, 0x03, 0x01}, uint16(len(h.raw))) // #nosec G115 -- bounded above
	record = append(record, h.raw...)
	spec, err := (&utls.Fingerprinter{}).FingerprintClientHello(record)
	if err != nil {
		return nil, fmt.Errorf("signal_client: %s: %w", h.Name, err)
	}
	if h.alpn != nil {
		found := false
		for _, ext := range spec.Extensions {
			if alpn, ok := ext.(*utls.ALPNExtension); ok {
				alpn.AlpnProtocols = append([]string(nil), h.alpn...)
				found = true
			}
		}
		if !found {
			return nil, fmt.Errorf("%w: %s", errNoALPN, h.Name)
		}
	}
	return spec, nil
}

// ALPN returns the protocols the hello offers.
func (h Hello) ALPN() []string {
	spec, err := h.Spec()
	if err != nil {
		return nil
	}
	for _, ext := range spec.Extensions {
		if alpn, ok := ext.(*utls.ALPNExtension); ok {
			return alpn.AlpnProtocols
		}
	}
	return nil
}

// signalRoots returns Signal's two roots (RSA and Ed25519), the only trust anchors Signal's
// clients accept for Signal-hosted services.
func signalRoots() (*x509.CertPool, error) {
	pool := x509.NewCertPool()
	for _, der := range [][]byte{signalRSARootDER, signalEd25519RootDER} {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("signal_client: parsing Signal root: %w", err)
		}
		pool.AddCert(cert)
	}
	return pool, nil
}
