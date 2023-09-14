package netxlite

//
// Code to ensure we log round trips
//

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// httpTransportLogger is an HTTPTransport with logging.
type httpTransportLogger struct {
	// HTTPTransport is the underlying HTTP transport.
	HTTPTransport model.HTTPTransport

	// Logger is the underlying logger.
	Logger model.DebugLogger
}

var _ model.HTTPTransport = &httpTransportLogger{}

func (txp *httpTransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	txp.Logger.Debugf("> %s %s", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			txp.Logger.Debugf("> %s: %s", key, value)
		}
	}
	txp.Logger.Debug(">")
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		txp.Logger.Debugf("< %s", err)
		return nil, err
	}
	txp.Logger.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			txp.Logger.Debugf("< %s: %s", key, value)
		}
	}
	txp.Logger.Debug("<")
	return resp, nil
}

func (txp *httpTransportLogger) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

func (txp *httpTransportLogger) Network() string {
	return txp.HTTPTransport.Network()
}
