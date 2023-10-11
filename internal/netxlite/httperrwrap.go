package netxlite

//
// Code to ensure we wrap errors
//

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// httpTransportErrWrapper is an HTTPTransport with error wrapping.
type httpTransportErrWrapper struct {
	HTTPTransport model.HTTPTransport
}

var _ model.HTTPTransport = &httpTransportErrWrapper{}

func (txp *httpTransportErrWrapper) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		return nil, NewTopLevelGenericErrWrapper(err)
	}
	return resp, nil
}

func (txp *httpTransportErrWrapper) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}

func (txp *httpTransportErrWrapper) Network() string {
	return txp.HTTPTransport.Network()
}

type httpClientErrWrapper struct {
	HTTPClient model.HTTPClient
}

func (c *httpClientErrWrapper) Do(req *http.Request) (*http.Response, error) {
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, NewTopLevelGenericErrWrapper(err)
	}
	return resp, nil
}

func (c *httpClientErrWrapper) CloseIdleConnections() {
	c.HTTPClient.CloseIdleConnections()
}
