package measurex

//
// DNSX (DNS eXtensions)
//
// This file contains basic networking code. We wrap the fundamental
// dnsx.RoundTripper type to store measurements into an EventDB.
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// DNSTransport is a transport for sending raw DNS queries
// and receiving raw DNS replies. The internal/netxlite/dnsx
// package implements a bunch of these transports.
type DNSTransport = dnsx.RoundTripper

// WrapDNSXRoundTripper wraps a dnsx.RoundTripper and returns a
// DNSTransport that saves DNSRoundTripEvents into the DB.
func WrapDNSXRoundTripper(
	measurementID int64, origin Origin, db EventDB, rt dnsx.RoundTripper) DNSTransport {
	return &dnsxTransportx{
		db:           db,
		RoundTripper: rt,
		origin:       origin,
		mid:          measurementID,
	}
}

type dnsxTransportx struct {
	dnsx.RoundTripper
	db     EventDB
	mid    int64
	origin Origin
}

// DNSRoundTripEvent contains the result of a DNS round trip. These
// events are generated by DNSTransport types.
type DNSRoundTripEvent struct {
	Origin        Origin        // OriginProbe or OriginTH
	MeasurementID int64         // ID of the measurement
	ConnID        int64         // connID (typically zero)
	Network       string        // DNS resolver's network (e.g., "dot", "doh")
	Address       string        // DNS resolver's address or URL (for "doh")
	Query         []byte        // Raw query
	Started       time.Duration // When we started the round trip
	Finished      time.Duration // When we were done
	Error         error         // Error or nil
	Reply         []byte        // Raw reply
}

func (txp *dnsxTransportx) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := txp.db.ElapsedTime()
	reply, err := txp.RoundTripper.RoundTrip(ctx, query)
	finished := txp.db.ElapsedTime()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		Origin:        txp.origin,
		MeasurementID: txp.mid,
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
