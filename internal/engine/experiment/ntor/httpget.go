package ntor

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/measuring/httptransport"
)

// doHTTPGet gets `/tor/status-vote/current/consensus.z`. We perform
// this action only for "dir_port" targets.
func (svc *service) doHTTPGet(ctx context.Context, out *serviceOutput, conn net.Conn) {
	defer conn.Close() // we own it
	URL := &url.URL{
		Scheme: "http",
		Host:   out.results.TargetAddress,
		Path:   "/tor/status-vote/current/consensus.z",
	}
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	if err != nil {
		out.err = err
		out.operation = "http_round_trip"
		return
	}
	resp, err := svc.httpTransport.RoundTrip(ctx, &httptransport.RoundTripRequest{
		Req:    req,
		Conn:   conn,
		Logger: svc.logger,
		Saver:  &out.saver,
	})
	if err != nil {
		out.err = err
		out.operation = "http_round_trip"
		return
	}
	defer resp.Body.Close()
	// TODO(bassosimone): don't read the whole body maybe?
	data, err := iox.ReadAllContext(ctx, resp.Body)
	if resp.Close && errors.Is(err, io.EOF) {
		err = nil
	}
	if err != nil {
		out.err = err
		out.operation = "http_read_body"
		return
	}
	out.body = data
}
