package measurex

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// DNSTransport is the DNS transport type we use.
type DNSTransport = dnsx.RoundTripper

// WrapDNSXRoundTripper wraps a dnsx.RoundTripper to add measurex capabilities.
func WrapDNSXRoundTripper(db DB, rt dnsx.RoundTripper) DNSTransport {
	return &dnsxTransportx{db: db, RoundTripper: rt}
}

type dnsxTransportx struct {
	dnsx.RoundTripper
	db DB
}

// DNSRoundTripEvent contains the result of a DNS round trip.
type DNSRoundTripEvent struct {
	MeasurementID int64
	Network       string
	Address       string
	Query         []byte
	Started       time.Time
	Finished      time.Time
	Error         error
	Reply         []byte
}

func (txp *dnsxTransportx) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Now()
	reply, err := txp.RoundTripper.RoundTrip(ctx, query)
	finished := time.Now()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		MeasurementID: txp.db.MeasurementID(),
		Network:       txp.RoundTripper.Network(),
		Address:       txp.RoundTripper.Address(),
		Query:         query,
		Started:       started,
		Finished:      finished,
		Error:         err,
		Reply:         reply,
	})
	return reply, err
}
