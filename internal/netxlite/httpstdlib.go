package netxlite

//
// Code to adapt oohttp to the stdlib and the stdlib to our HTTP models
//

import (
	"net/http"

	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// stdlibTransport wraps oohttp.StdlibTransport to add .Network()
type httpTransportStdlib struct {
	StdlibTransport *oohttp.StdlibTransport
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
