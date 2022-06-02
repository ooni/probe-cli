package filtering

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"net"
	"net/http"
	"net/url"

	"github.com/google/martian/v3/mitm"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPAction is an HTTP filtering action that this server should take.
type HTTPAction string

const (
	// HTTPActionReset resets the connection.
	HTTPActionReset = HTTPAction("reset")

	// HTTPActionTimeout causes the connection to timeout.
	HTTPActionTimeout = HTTPAction("timeout")

	// HTTPActionEOF causes the connection to EOF.
	HTTPActionEOF = HTTPAction("eof")

	// HTTPAction451 causes the proxy to return a 451 error.
	HTTPAction451 = HTTPAction("451")

	// HTTPActionDoH causes the proxy to return a sensible reply
	// with static IP addresses if the request is DoH.
	HTTPActionDoH = HTTPAction("doh")
)

// HTTPServer is a server that implements filtering policies.
type HTTPServer struct {
	// action is the action to implement.
	action HTTPAction

	// cert is the fake CA certificate.
	cert *x509.Certificate

	// config is the config to generate certificates on the fly.
	config *mitm.Config

	// privkey is the private key that signed the cert.
	privkey *rsa.PrivateKey

	// server is the underlying server.
	server *http.Server

	// url contains the server URL
	url *url.URL
}

// NewHTTPServerCleartext creates a new HTTPServer using cleartext HTTP.
func NewHTTPServerCleartext(action HTTPAction) *HTTPServer {
	return newHTTPOrHTTPSServer(action, false)
}

// NewHTTPServerTLS creates a new HTTP server using HTTPS.
func NewHTTPServerTLS(action HTTPAction) *HTTPServer {
	return newHTTPOrHTTPSServer(action, true)
}

// Close closes the server ASAP.
func (p *HTTPServer) Close() error {
	return p.server.Close()
}

// URL returns the server's URL
func (p *HTTPServer) URL() *url.URL {
	return p.url
}

// TLSConfig returns a suitable base TLS config for the client.
func (p *HTTPServer) TLSConfig() *tls.Config {
	config := &tls.Config{}
	if p.cert != nil {
		o := x509.NewCertPool()
		o.AddCert(p.cert)
		config.RootCAs = o
	}
	return config
}

// newHTTPOrHTTPSServer is an internal factory for creating a new instance.
func newHTTPOrHTTPSServer(action HTTPAction, enableTLS bool) *HTTPServer {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	runtimex.PanicOnError(err, "net.Listen failed")
	srv := &HTTPServer{
		action:  action,
		cert:    nil,
		config:  nil,
		privkey: nil,
		server:  nil,
		url: &url.URL{
			Scheme: "",
			Host:   listener.Addr().String(),
		},
	}
	srv.server = &http.Server{Handler: srv}
	switch enableTLS {
	case false:
		srv.url.Scheme = "http"
		go srv.server.Serve(listener)
	case true:
		srv.url.Scheme = "https"
		srv.cert, srv.privkey, srv.config = tlsConfigMITM()
		srv.server.TLSConfig = srv.config.TLS()
		go srv.server.ServeTLS(listener, "", "") // using server.TLSConfig
	}
	return srv
}

// HTTPBlockPage451 is the block page returned along with status 451
var HTTPBlockpage451 = []byte(`<html><head>
  <title>451 Unavailable For Legal Reasons</title>
</head><body>
  <center><h1>451 Unavailable For Legal Reasons</h1></center>
  <p>This content is not available in your jurisdiction.</p>
</body></html>
`)

// ServeHTTP serves HTTP requests
func (p *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch p.action {
	case HTTPActionReset, HTTPActionTimeout, HTTPActionEOF:
		p.hijack(w, r, p.action)
	case HTTPAction451:
		w.WriteHeader(http.StatusUnavailableForLegalReasons)
		w.Write(HTTPBlockpage451)
	case HTTPActionDoH:
		p.doh(w, r)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (p *HTTPServer) hijack(w http.ResponseWriter, r *http.Request, policy HTTPAction) {
	// Note:
	//
	// 1. we assume we can hihack the connection
	//
	// 2. Hijack won't fail the first time it's invoked
	hijacker := w.(http.Hijacker)
	conn, _, err := hijacker.Hijack()
	runtimex.PanicOnError(err, "hijacker.Hijack failed")
	defer conn.Close()
	switch policy {
	case HTTPActionReset:
		if tc, ok := conn.(*net.TCPConn); ok {
			tc.SetLinger(0)
		}
	case HTTPActionTimeout:
		<-r.Context().Done()
	case HTTPActionEOF:
		// nothing
	}
}

func (p *HTTPServer) doh(w http.ResponseWriter, r *http.Request) {
	rawQuery, err := netxlite.ReadAllContext(r.Context(), r.Body)
	if err != nil {
		w.WriteHeader(400)
		return
	}
	query := &dns.Msg{}
	if err := query.Unpack(rawQuery); err != nil {
		w.WriteHeader(400)
		return
	}
	if query.Response {
		w.WriteHeader(400)
		return
	}
	response := dnsCompose(query, net.IPv4(8, 8, 8, 8), net.IPv4(8, 8, 4, 4))
	rawResponse, err := response.Pack()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	w.Write(rawResponse)
}
