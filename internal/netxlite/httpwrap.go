package netxlite

//
// High level public wrappers
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// WrapHTTPTransport creates an HTTPTransport using the given logger
// and guarantees that returned errors are wrapped.
//
// This is a low level factory. Consider not using it directly.
func WrapHTTPTransport(logger model.DebugLogger, txp model.HTTPTransport) model.HTTPTransport {
	return &httpTransportLogger{
		HTTPTransport: &httpTransportErrWrapper{txp},
		Logger:        logger,
	}
}

// WrapHTTPClient wraps an HTTP client to add error wrapping capabilities.
func WrapHTTPClient(clnt model.HTTPClient) model.HTTPClient {
	return &httpClientErrWrapper{clnt}
}
