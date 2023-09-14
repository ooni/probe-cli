package netxlite

import (
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewHTTPClient creates a new, wrapped HTTPClient using the given transport.
func NewHTTPClient(txp model.HTTPTransport) model.HTTPClient {
	return WrapHTTPClient(&http.Client{Transport: txp})
}
