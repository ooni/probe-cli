package testingx

import (
	"io"
	"net/http"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// HTTPProxyHandlerNetx abstracts [*netxlite.Netx] for the [*HTTPProxyHandler].
type HTTPProxyHandlerNetx interface {
	// NewDialerWithResolver creates a new dialer using the given resolver and logger.
	NewDialerWithResolver(dl model.DebugLogger, r model.Resolver, w ...model.DialerWrapper) model.Dialer

	// NewHTTPTransportStdlib creates a new HTTP transport using the stdlib.
	NewHTTPTransportStdlib(dl model.DebugLogger) model.HTTPTransport

	// NewStdlibResolver creates a new resolver that tries to use the getaddrinfo libc call.
	NewStdlibResolver(logger model.DebugLogger) model.Resolver
}

// httpProxyHandler is an HTTP/HTTPS proxy.
type httpProxyHandler struct {
	// Logger is the logger to use.
	Logger model.Logger

	// Netx is the network to use.
	Netx HTTPProxyHandlerNetx
}

// NewHTTPProxyHandler constructs a new [*HTTPProxyHandler].
func NewHTTPProxyHandler(logger model.Logger, netx HTTPProxyHandlerNetx) http.Handler {
	return &httpProxyHandler{
		Logger: &logx.PrefixLogger{
			Prefix: "PROXY: ",
			Logger: logger,
		},
		Netx: netx,
	}
}

// ServeHTTP implements http.Handler.
func (ph *httpProxyHandler) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	ph.Logger.Infof("request: %+v", req)

	switch req.Method {
	case http.MethodConnect:
		ph.connect(rw, req)

	case http.MethodGet:
		ph.get(rw, req)

	default:
		rw.WriteHeader(http.StatusNotImplemented)
	}
}

func (ph *httpProxyHandler) connect(rw http.ResponseWriter, req *http.Request) {
	resolver := ph.Netx.NewStdlibResolver(ph.Logger)
	dialer := ph.Netx.NewDialerWithResolver(ph.Logger, resolver)

	sconn, err := dialer.DialContext(req.Context(), "tcp", req.Host)
	if err != nil {
		rw.WriteHeader(http.StatusBadGateway)
		return
	}
	defer sconn.Close()

	hijacker := rw.(http.Hijacker)
	cconn, buffered := runtimex.Try2(hijacker.Hijack())
	runtimex.Assert(buffered.Reader.Buffered() <= 0, "data before finishing HTTP handshake")
	defer cconn.Close()

	_, _ = cconn.Write([]byte("HTTP/1.1 200 Ok\r\n\r\n"))

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(sconn, cconn)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		_, _ = io.Copy(cconn, sconn)
	}()

	wg.Wait()
}

func (ph *httpProxyHandler) get(rw http.ResponseWriter, req *http.Request) {
	// reject requests that already visited the proxy and requests we cannot route
	if req.Host == "" || req.Header.Get("Via") != "" {
		rw.WriteHeader(http.StatusBadRequest)
		return
	}

	// clone the request before modifying it
	req = req.Clone(req.Context())

	// include proxy header to prevent sending requests to ourself
	req.Header.Add("Via", "testingx/0.1.0")

	// fix: "http: Request.RequestURI can't be set in client requests"
	req.RequestURI = ""

	// fix: `http: unsupported protocol scheme ""`
	req.URL.Host = req.Host

	// fix: "http: no Host in request URL"
	req.URL.Scheme = "http"

	ph.Logger.Debugf("sending request: %s", req)

	// create HTTP client using netx
	txp := ph.Netx.NewHTTPTransportStdlib(ph.Logger)
	defer txp.CloseIdleConnections()

	// obtain response
	resp, err := txp.RoundTrip(req)
	if err != nil {
		ph.Logger.Warnf("request failed: %s", err.Error())
		rw.WriteHeader(http.StatusBadGateway)
		return
	}

	// write response
	rw.WriteHeader(resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			rw.Header().Add(key, value)
		}
	}

	// write response body
	_, _ = io.Copy(rw, resp.Body)
}
