package iplookup

import (
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type FakeTransport struct {
	Err  error
	Resp *http.Response
}

func (txp FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	if req.Body != nil {
		netxlite.ReadAllContext(req.Context(), req.Body)
		req.Body.Close()
	}
	if txp.Err != nil {
		return nil, txp.Err
	}
	txp.Resp.Request = req // non thread safe but it doesn't matter
	return txp.Resp, nil
}

func (txp FakeTransport) CloseIdleConnections() {}
