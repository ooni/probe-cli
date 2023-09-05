package testingx

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPServer is a server tuned for testing that works with both the
// standard library and netem as its net backend. The zero value of this
// struct is invalid; please, use the appropriate constructor.
//
// This struct tries to mimic [*net/http/httptest.Server] to simplify
// transitioning the code from that struct to this one.
type HTTPServer struct {
	// Config contains the server started by the constructor.
	Config *http.Server

	// Listener is the underlying [net.Listener].
	Listener net.Listener

	// TLS contains the TLS configuration used by the constructor, or nil
	// if you constructed a server that does not use TLS.
	TLS *tls.Config

	// URL is the base URL used by the server.
	URL string

	// X509CertPool is the X.509 cert pool we're using or nil.
	X509CertPool *x509.CertPool
}

// MustNewHTTPServer is morally equivalent to [httptest.NewHTTPServer].
func MustNewHTTPServer(handler http.Handler) *HTTPServer {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	return MustNewHTTPServerEx(addr, &TCPListenerStdlib{}, handler)
}

// MustNewHTTPServerTLS is morally equivalent to [httptest.NewHTTPServerTLS].
func MustNewHTTPServerTLS(handler http.Handler) *HTTPServer {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	provider := MustNewTLSMITMProviderNetem()
	return MustNewHTTPServerTLSEx(addr, &TCPListenerStdlib{}, handler, provider)
}

// MustNewHTTPServerEx creates a new [HTTPServer] using HTTP or PANICS.
func MustNewHTTPServerEx(addr *net.TCPAddr, listener TCPListener, handler http.Handler) *HTTPServer {
	return mustNewHTTPServer(addr, listener, handler, optional.None[TLSMITMProvider]())
}

// MustNewHTTPServerTLSEx creates a new [HTTPServer] using HTTPS or PANICS.
func MustNewHTTPServerTLSEx(addr *net.TCPAddr, listener TCPListener, handler http.Handler, mitm TLSMITMProvider) *HTTPServer {
	return mustNewHTTPServer(addr, listener, handler, optional.Some(mitm))
}

// newHTTPOrHTTPSServer is an internal factory for creating a new instance.
func mustNewHTTPServer(
	addr *net.TCPAddr,
	httpListener TCPListener,
	handler http.Handler,
	tlsConfig optional.Value[TLSMITMProvider],
) *HTTPServer {
	listener := runtimex.Try1(httpListener.ListenTCP("tcp", addr))
	srv := &HTTPServer{
		Config:       &http.Server{Handler: handler},
		Listener:     listener,
		TLS:          nil, // the default when not using TLS
		URL:          "",  // filled later
		X509CertPool: nil, // the default when not using TLS
	}
	baseURL := &url.URL{Host: listener.Addr().String()}
	switch !tlsConfig.IsNone() {
	case true:
		baseURL.Scheme = "https"
		srv.TLS = tlsConfig.Unwrap().ServerTLSConfig()
		srv.Config.TLSConfig = srv.TLS
		srv.X509CertPool = runtimex.Try1(tlsConfig.Unwrap().DefaultCertPool())
		go srv.Config.ServeTLS(listener, "", "") // using server.TLSConfig
	default:
		baseURL.Scheme = "http"
		go srv.Config.Serve(listener)
	}
	srv.URL = baseURL.String()
	return srv
}

// Close closes the server as soon as possibile.
func (p *HTTPServer) Close() error {
	return p.Config.Close()
}

// HTTPBlockPage451 is the block page returned along with status 451
var HTTPBlockpage451 = []byte(`<html><head>
  <title>451 Unavailable For Legal Reasons</title>
</head><body>
  <center><h1>451 Unavailable For Legal Reasons</h1></center>
  <p>This content is not available in your jurisdiction.</p>
</body></html>
`)

// HTTPHandlerBlockpage451 returns a handler that returns 451 along with a blockpage.
func HTTPHandlerBlockpage451() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnavailableForLegalReasons)
		w.Write(HTTPBlockpage451)
	})
}

// HTTPHandlerEOF returns a handler that immediately closes the connection.
func HTTPHandlerEOF() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpHandlerHijack(w, r, "eof")
	})
}

// HTTPHandlerReset returns a handler that immediately resets the connection.
//
// Bug: this handler does not WAI when using [github.com/ooni/netem]. The reason why this happens
// is that gvisor.io supports SO_LINGER but there's no *gonet.TCPConn.SetLinger.
func HTTPHandlerReset() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpHandlerHijack(w, r, "reset")
	})
}

// HTTPHandlerTimeout returns a handler that never returns a response and instead
// blocks on the request context, thus causing a client timeout.
func HTTPHandlerTimeout() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		httpHandlerHijack(w, r, "timeout")
	})
}

func httpHandlerHijack(w http.ResponseWriter, r *http.Request, policy string) {
	// Note:
	//
	// 1. we assume we can hihack the connection
	//
	// 2. Hijack won't fail the first time it's invoked
	hijacker := w.(http.Hijacker)
	conn, _ := runtimex.Try2(hijacker.Hijack())

	defer conn.Close()

	switch policy {
	case "reset":
		tcpMaybeResetNetConn(conn)

	case "timeout":
		<-r.Context().Done()

	case "eof":
		// nothing
	}
}
