package netxlite

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// stdlibTransport wraps a httpTransportStdlib to add .Network()
type httpTransportStdlib struct {
	StdlibTransport *http.Transport
}

var _ model.HTTPTransport = &httpTransportStdlib{}

func (txp *httpTransportStdlib) CloseIdleConnections() {
	txp.StdlibTransport.CloseIdleConnections()
}

func (txp *httpTransportStdlib) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.StdlibTransport.RoundTrip(req)
}

// Network implements HTTPTransport.Network.
func (txp *httpTransportStdlib) Network() string {
	return "tcp"
}
