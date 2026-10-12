package signalclient

//
// The TLS ClientHellos of real Signal clients, reproduced byte for byte with uTLS.
//
// Each file in hellos/ is a ClientHello handshake message (starting with type 1) captured from
// the client's own network code; see hellos/README.md for how each was captured. We send the
// same hello the real client sends so that censorship keyed on Signal's fingerprint is measured.
//

import (
	_ "embed"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"strings"

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

	// maxTLS12 limits a captured hello to TLS 1.2 (Signal-Android's Google fronts): the TLS 1.3
	// extensions (supported_versions, key_share, psk_key_exchange_modes) are removed.
	maxTLS12 bool

	// parrot, when not nil, is a uTLS browser parrot used instead of a captured hello (the
	// browser_hello control).
	parrot *utls.ClientHelloID

	// stdlibTLS12 means Go's crypto/tls limited to TLS 1.2, offering alpn (the tls12 control).
	stdlibTLS12 bool
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

// HelloChrome is uTLS's current Chrome parrot, a browser's hello for the browser_hello control.
var HelloChrome = Hello{Name: "utls-" + strings.ToLower(utls.HelloChrome_Auto.Client) + "-" + utls.HelloChrome_Auto.Version,
	parrot: &utls.HelloChrome_Auto}

// helloStdlibTLS12 is Go's TLS client limited to TLS 1.2, offering the ALPN of like. The ALPN is
// part of the name ("go-crypto-tls-1.2+h2,http/1.1"), so that different hellos have different names.
func helloStdlibTLS12(like Hello) Hello {
	alpn := like.ALPN()
	name := "go-crypto-tls-1.2"
	if len(alpn) > 0 {
		name += "+" + strings.Join(alpn, ",")
	}
	return Hello{Name: name, stdlibTLS12: true, alpn: alpn}
}

// limitedToTLS12 is h limited to TLS 1.2; its name says so.
func (h Hello) limitedToTLS12() Hello {
	h.Name += "-tls12"
	h.maxTLS12 = true
	return h
}

// errNoALPN means a hello has no ALPN extension to replace.
var errNoALPN = errors.New("signal_client: captured hello has no ALPN extension")

// Spec returns a uTLS spec that reproduces the captured hello (or the parrot's spec).
func (h Hello) Spec() (*utls.ClientHelloSpec, error) {
	if h.parrot != nil {
		spec, err := utls.UTLSIdToSpec(*h.parrot)
		return &spec, err
	}
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
	if h.maxTLS12 {
		var kept []utls.TLSExtension
		for _, ext := range spec.Extensions {
			switch ext.(type) {
			case *utls.SupportedVersionsExtension, *utls.KeyShareExtension, *utls.PSKKeyExchangeModesExtension:
				continue
			}
			kept = append(kept, ext)
		}
		spec.Extensions = kept
		spec.TLSVersMin, spec.TLSVersMax = utls.VersionTLS12, utls.VersionTLS12
	}
	return spec, nil
}

// ALPN returns the protocols the hello offers.
func (h Hello) ALPN() []string {
	if h.stdlibTLS12 {
		return h.alpn
	}
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
