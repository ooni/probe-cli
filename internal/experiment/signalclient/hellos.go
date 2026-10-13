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

	//go:embed hellos/libsignal-reflector.bin
	rawHelloLibsignalReflector []byte

	//go:embed hellos/android-okhttp-websocket.bin
	rawHelloAndroidWebSocket []byte

	//go:embed hellos/android-okhttp-rest.bin
	rawHelloAndroidREST []byte

	//go:embed hellos/android-okhttp-front-g-gmail-websocket.bin
	rawHelloAndroidGmailWebSocket []byte

	//go:embed hellos/android-okhttp-front-g-gmail-rest.bin
	rawHelloAndroidGmailREST []byte

	//go:embed hellos/android-okhttp-front-g-gmaps-websocket.bin
	rawHelloAndroidGmapsWebSocket []byte

	//go:embed hellos/android-okhttp-front-g-gmaps-rest.bin
	rawHelloAndroidGmapsREST []byte

	//go:embed hellos/android-okhttp-front-f-websocket.bin
	rawHelloAndroidModernWebSocket []byte

	//go:embed hellos/android-okhttp-front-f-rest.bin
	rawHelloAndroidModernREST []byte

	//go:embed hellos/desktop-electron.bin
	rawHelloDesktop []byte

	//go:embed hellos/signal.cer
	signalRSARootDER []byte

	//go:embed hellos/signal-ed25519.cer
	signalEd25519RootDER []byte
)

// The client versions the hellos were captured from (hellos/README.md).
const (
	libsignalVersion = "0.105.0"
	androidVersion   = "8.31.1"
	okhttpConscrypt  = "okhttp-5.3.2-conscrypt-2.6.2"
	desktopVersion   = "8.33.0"
	electronVersion  = "44.4.5"
)

// Client is the Signal client whose connection a target reproduces: its User-Agent.
type Client struct {
	userAgent string // the real client's (sent with Config.AppUserAgent)
	libsignal bool   // libsignal appends " libsignal/<version>" to its User-Agent
}

// The clients whose connections the targets reproduce.
var (
	ClientAndroid   = Client{userAgent: "Signal-Android/" + androidVersion + " Android/34"}
	ClientLibsignal = Client{userAgent: ClientAndroid.userAgent + " libsignal/" + libsignalVersion, libsignal: true}
	ClientDesktop   = Client{userAgent: "Signal-Desktop/" + desktopVersion + " Linux"}
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

	// parrot, when not nil, is a uTLS browser parrot used instead of a captured hello (the
	// browser_hello control).
	parrot *utls.ClientHelloID

	// stdlibTLS12 means Go's crypto/tls limited to TLS 1.2, offering alpn (the tls12 control).
	stdlibTLS12 bool
}

var (
	// HelloLibsignal is libsignal 0.105.0's direct-route hello, the version Signal-Android and
	// Signal-iOS pin (chat on both, provisioning on iOS): TLS 1.3 only, groups x25519, P-256 and
	// P-384 with an x25519 key share, Ed25519 first, ALPN h2. Signal-Desktop's libsignal 0.103.0
	// sends the same hello (captured separately on 2026-10-12).
	HelloLibsignal = Hello{Name: "libsignal-" + libsignalVersion, raw: rawHelloLibsignal}

	// HelloLibsignalEnclave is the same hello with ALPN http/1.1, which libsignal uses for the
	// CDSI, SVR2 and SVR-B enclaves (same ssl_config, alpn parameter differs; net/src/env.rs).
	HelloLibsignalEnclave = Hello{Name: "libsignal-" + libsignalVersion + "-enclave", raw: rawHelloLibsignal, alpn: []string{"http/1.1"}}

	// HelloLibsignalReflector is libsignal 0.105.0's hello to a front on the reflector routes
	// (proxyf, proxyg), the same on all eight fronts: TLS 1.2 and 1.3, ALPN http/1.1, padded to
	// 512 bytes. Inside the tunnel, libsignal sends its usual hello to the target.
	HelloLibsignalReflector = Hello{Name: "libsignal-" + libsignalVersion + "-reflector", raw: rawHelloLibsignalReflector}

	// HelloAndroidWebSocket is Signal-Android 8.31.1's OkHttp 5.3.2 + Conscrypt 2.6.2 hello for
	// WebSockets (provisioning): ALPN http/1.1, session tickets, X25519MLKEM768 key share (1477 bytes).
	HelloAndroidWebSocket = Hello{Name: "android-" + okhttpConscrypt + "-websocket", raw: rawHelloAndroidWebSocket}

	// HelloAndroidREST is the same client's hello for HTTPS (storage, CDNs): ALPN h2,http/1.1.
	HelloAndroidREST = Hello{Name: "android-" + okhttpConscrypt + "", raw: rawHelloAndroidREST}

	// Signal-Android's hellos to its own domain fronts (SignalServiceNetworkAccess.kt), the same
	// client with each front's ConnectionSpec, for WebSockets (provisioning) and HTTPS (storage):
	//   - Gmail: GMAIL_CONNECTION_SPEC, and PLAY_CONNECTION_SPEC (the same list), for www.google.com,
	//     googlemail.com, android.clients.google.com and the countries' Google domains: TLS 1.2
	//     only, 9 cipher suites;
	//   - Maps: GMAPS_CONNECTION_SPEC, for clients3 and clients4.google.com: TLS 1.2 only, 14 suites;
	//   - Modern: ConnectionSpec.MODERN_TLS, for the Fastly fronts: TLS 1.2 and 1.3, X25519MLKEM768.
	HelloAndroidGmailWebSocket = Hello{Name: "android-" + okhttpConscrypt + "-gmail-websocket",
		raw: rawHelloAndroidGmailWebSocket}
	HelloAndroidGmailREST = Hello{Name: "android-" + okhttpConscrypt + "-gmail",
		raw: rawHelloAndroidGmailREST}
	HelloAndroidGmapsWebSocket = Hello{Name: "android-" + okhttpConscrypt + "-gmaps-websocket",
		raw: rawHelloAndroidGmapsWebSocket}
	HelloAndroidGmapsREST = Hello{Name: "android-" + okhttpConscrypt + "-gmaps",
		raw: rawHelloAndroidGmapsREST}
	HelloAndroidModernWebSocket = Hello{Name: "android-" + okhttpConscrypt + "-modern-websocket",
		raw: rawHelloAndroidModernWebSocket}
	HelloAndroidModernREST = Hello{Name: "android-" + okhttpConscrypt + "-modern",
		raw: rawHelloAndroidModernREST}

	// HelloDesktop is Signal-Desktop's own fetches (updater, WebAPI's agent) through Electron
	// 44.4.5's Node and BoringSSL: no ALPN, X25519MLKEM768 + x25519 key shares (1475 bytes).
	HelloDesktop = Hello{Name: "desktop-electron-" + electronVersion, raw: rawHelloDesktop}
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
	return spec, nil
}

// apply sets uconn up to send the hello: its uTLS spec and, where the captured hello sends an
// empty legacy session id (Signal-Android's TLS 1.2-only Google-front hellos), an empty one too.
// ApplyPreset always fills it with 32 random bytes, as TLS 1.3 clients send.
func (h Hello) apply(uconn *utls.UConn) error {
	spec, err := h.Spec()
	if err != nil {
		return err
	}
	if err := uconn.ApplyPreset(spec); err != nil {
		return err
	}
	if h.emptySessionID() {
		uconn.HandshakeState.Hello.SessionId = nil
	}
	return nil
}

// emptySessionID reports whether the captured hello's legacy session id is empty. In the
// handshake message, its length follows the type, length, version and random (1+3+2+32 bytes).
func (h Hello) emptySessionID() bool {
	return h.parrot == nil && len(h.raw) > 38 && h.raw[38] == 0
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
