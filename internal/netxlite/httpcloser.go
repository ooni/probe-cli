package netxlite

//
// Code to ensure we forward CloseIdleConnection calls
//

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// httpTransportConnectionsCloser is an HTTPTransport that
// correctly forwards CloseIdleConnections calls.
type httpTransportConnectionsCloser struct {
	HTTPTransport model.HTTPTransport
	Dialer        model.Dialer
	TLSDialer     model.TLSDialer
}

var _ model.HTTPTransport = &httpTransportConnectionsCloser{}

func (txp *httpTransportConnectionsCloser) RoundTrip(req *http.Request) (*http.Response, error) {
	return txp.HTTPTransport.RoundTrip(req)
}

func (txp *httpTransportConnectionsCloser) Network() string {
	return txp.HTTPTransport.Network()
}

// CloseIdleConnections forwards the CloseIdleConnections calls.
func (txp *httpTransportConnectionsCloser) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
	txp.Dialer.CloseIdleConnections()
	txp.TLSDialer.CloseIdleConnections()
}
