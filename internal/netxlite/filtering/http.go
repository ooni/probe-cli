package filtering

import (
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPAction is an HTTP filtering action that this proxy should take.
type HTTPAction string

const (
	// HTTPActionPass passes the traffic to the destination.
	HTTPActionPass = HTTPAction("pass")

	// HTTPActionReset resets the connection.
	HTTPActionReset = HTTPAction("reset")

	// HTTPActionTimeout causes the connection to timeout.
	HTTPActionTimeout = HTTPAction("timeout")

	// HTTPActionEOF causes the connection to EOF.
	HTTPActionEOF = HTTPAction("eof")

	// HTTPAction451 causes the proxy to return a 451 error.
	HTTPAction451 = HTTPAction("451")
)

// HTTPProxy is a proxy that routes traffic depending on the
// host header and may implement filtering policies.
type HTTPProxy struct {
	// OnIncomingHost is the MANDATORY hook called whenever we have
	// successfully received an HTTP request.
	OnIncomingHost func(host string) HTTPAction
}

// Start starts the proxy.
func (p *HTTPProxy) Start(address string) (net.Listener, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}
	server := &http.Server{Handler: p}
	go server.Serve(listener)
	return listener, nil
}

var httpBlockpage451 = []byte(`<html><head>
  <title>451 Unavailable For Legal Reasons</title>
</head><body>
  <center><h1>451 Unavailable For Legal Reasons</h1></center>
  <p>This content is not available in your jurisdiction.</p>
</body></html>
`)

const httpProxyProduct = "jafar/0.1.0"

// ServeHTTP serves HTTP requests
func (p *HTTPProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Implementation note: use Via header to detect in a loose way
	// requests originated by us and directed to us.
	if r.Header.Get("Via") == httpProxyProduct || r.Host == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	p.handle(w, r)
}

func (p *HTTPProxy) handle(w http.ResponseWriter, r *http.Request) {
	switch policy := p.OnIncomingHost(r.Host); policy {
	case HTTPActionPass:
		p.proxy(w, r)
	case HTTPActionReset, HTTPActionTimeout, HTTPActionEOF:
		p.hijack(w, r, policy)
	case HTTPAction451:
		w.WriteHeader(http.StatusUnavailableForLegalReasons)
		w.Write(httpBlockpage451)
	default:
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func (p *HTTPProxy) hijack(w http.ResponseWriter, r *http.Request, policy HTTPAction) {
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

func (p *HTTPProxy) proxy(w http.ResponseWriter, r *http.Request) {
	r.Header.Add("Via", httpProxyProduct) // see ServeHTTP
	proxy := httputil.NewSingleHostReverseProxy(&url.URL{
		Host:   r.Host,
		Scheme: "http",
	})
	proxy.Transport = http.DefaultTransport
	proxy.ServeHTTP(w, r)
}
