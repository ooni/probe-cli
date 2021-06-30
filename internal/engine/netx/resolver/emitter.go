package resolver

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
)

// EmitterTransport is a RoundTripper that emits events when they occur.
type EmitterTransport struct {
	RoundTripper
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp EmitterTransport) RoundTrip(ctx context.Context, querydata []byte) ([]byte, error) {
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(modelx.Measurement{
		DNSQuery: &modelx.DNSQueryEvent{
			Data:                   querydata,
			DurationSinceBeginning: time.Since(root.Beginning),
		},
	})
	replydata, err := txp.RoundTripper.RoundTrip(ctx, querydata)
	if err != nil {
		return nil, err
	}
	root.Handler.OnMeasurement(modelx.Measurement{
		DNSReply: &modelx.DNSReplyEvent{
			Data:                   replydata,
			DurationSinceBeginning: time.Since(root.Beginning),
		},
	})
	return replydata, nil
}

// EmitterResolver is a resolver that emits events
type EmitterResolver struct {
	Resolver
}

// LookupHost returns the IP addresses of a host
func (r EmitterResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	var (
		network string
		address string
	)
	type queryableResolver interface {
		Transport() RoundTripper
	}
	if qr, ok := r.Resolver.(queryableResolver); ok {
		txp := qr.Transport()
		network, address = txp.Network(), txp.Address()
	}
	root := modelx.ContextMeasurementRootOrDefault(ctx)
	root.Handler.OnMeasurement(modelx.Measurement{
		ResolveStart: &modelx.ResolveStartEvent{
			DurationSinceBeginning: time.Since(root.Beginning),
			Hostname:               hostname,
			TransportAddress:       address,
			TransportNetwork:       network,
		},
	})
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	root.Handler.OnMeasurement(modelx.Measurement{
		ResolveDone: &modelx.ResolveDoneEvent{
			Addresses:              addrs,
			DurationSinceBeginning: time.Since(root.Beginning),
			Error:                  err,
			Hostname:               hostname,
			TransportAddress:       address,
			TransportNetwork:       network,
		},
	})
	return addrs, err
}

var _ RoundTripper = EmitterTransport{}
var _ Resolver = EmitterResolver{}
