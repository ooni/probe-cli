package testingx

import (
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/randx"
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
	//
	// This field also exists in the [*net/http/httptest.Server] struct.
	Config *http.Server

	// Listener is the underlying [net.Listener].
	//
	// This field also exists in the [*net/http/httptest.Server] struct.
	Listener net.Listener

	// TLS contains the TLS configuration used by the constructor, or nil
	// if you constructed a server that does not use TLS.
	//
	// This field also exists in the [*net/http/httptest.Server] struct.
	TLS *tls.Config

	// URL is the base URL used by the server.
	//
	// This field also exists in the [*net/http/httptest.Server] struct.
	URL string

	// X509CertPool is the X.509 cert pool we're using or nil.
	//
	// This field is an extension that is not present in the httptest package.
	X509CertPool *x509.CertPool

	// CACert is the CA used by this server or nil.
	//
	// This field is an extension that is not present in the httptest package.
	CACert *x509.Certificate
}

// MustNewHTTPServer is morally equivalent to [httptest.NewHTTPServer].
func MustNewHTTPServer(handler http.Handler) *HTTPServer {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	return MustNewHTTPServerEx(addr, &TCPListenerStdlib{}, handler)
}

// MustNewHTTPServerEx creates a new [HTTPServer] using HTTP or PANICS.
func MustNewHTTPServerEx(addr *net.TCPAddr, httpListener TCPListener, handler http.Handler) *HTTPServer {
	listener := runtimex.Try1(httpListener.ListenTCP("tcp", addr))

	baseURL := &url.URL{
		Scheme: "http",
		Host:   listener.Addr().String(),
		Path:   "/",
	}
	srv := &HTTPServer{
		Config:       &http.Server{Handler: handler},
		Listener:     listener,
		TLS:          nil,
		URL:          baseURL.String(),
		X509CertPool: nil,
		CACert:       nil,
	}

	go srv.Config.Serve(listener)

	return srv
}

// MustNewHTTPServerTLS is morally equivalent to [httptest.NewHTTPServerTLS].
func MustNewHTTPServerTLS(
	handler http.Handler,
	ca netem.CertificationAuthority,
	commonName string,
	extraSNIs ...string,
) *HTTPServer {
	addr := &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0}
	return MustNewHTTPServerTLSEx(addr, &TCPListenerStdlib{}, handler, ca, commonName, extraSNIs...)
}

// MustNewHTTPServerTLSEx creates a new [HTTPServer] using HTTPS or PANICS.
func MustNewHTTPServerTLSEx(
	addr *net.TCPAddr,
	httpListener TCPListener,
	handler http.Handler,
	ca netem.CertificationAuthority,
	commonName string,
	extraSNIs ...string,
) *HTTPServer {
	listener := runtimex.Try1(httpListener.ListenTCP("tcp", addr))

	baseURL := &url.URL{
		Scheme: "https",
		Host:   listener.Addr().String(),
		Path:   "/",
	}

	otherNames := append([]string{}, addr.IP.String())
	otherNames = append(otherNames, extraSNIs...)

	srv := &HTTPServer{
		Config:       &http.Server{Handler: handler},
		Listener:     listener,
		TLS:          ca.MustNewServerTLSConfig(commonName, otherNames...),
		URL:          baseURL.String(),
		X509CertPool: ca.DefaultCertPool(),
		CACert:       ca.CACert(),
	}

	srv.Config.TLSConfig = srv.TLS
	go srv.Config.ServeTLS(listener, "", "") // using server.TLSConfig

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
	conn := httpHijack(w)
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

// HTTPHandlerResetWhileReadingBody returns a handler that sends a
// connection reset by peer while the client is reading the body.
func HTTPHandlerResetWhileReadingBody() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn := httpHijack(w)
		defer conn.Close()

		// write the HTTP response headers
		conn.Write([]byte("HTTP/1.1 200 Ok\r\n"))
		conn.Write([]byte("Content-Type: text/html\r\n"))
		conn.Write([]byte("Content-Length: 65535\r\n"))
		conn.Write([]byte("\r\n"))

		// start writing the response
		content := randx.Letters(32768)
		conn.Write([]byte(content))

		// sleep for half a second simulating something wrong
		time.Sleep(500 * time.Millisecond)

		// finally issue reset for the conn
		tcpMaybeResetNetConn(conn)
	})
}

// httpHijack is a convenience function to hijack the underlying connection.
func httpHijack(w http.ResponseWriter) net.Conn {
	// Note:
	//
	// 1. we assume we can hihack the connection
	//
	// 2. Hijack won't fail the first time it's invoked
	hijacker := w.(http.Hijacker)
	conn, _ := runtimex.Try2(hijacker.Hijack())
	return conn
}
