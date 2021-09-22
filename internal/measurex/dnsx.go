package measurex

//
// DNSX (DNS eXtensions)
//
// We wrap dnsx.RoundTripper to store events into a WritableDB.
//

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
)

// DNSXRoundTripper is a transport for sending raw DNS queries
// and receiving raw DNS replies. The internal/netxlite/dnsx
// package implements a bunch of these transports.
type DNSXRoundTripper = dnsx.RoundTripper

// WrapDNSXRoundTripper creates a new DNSXRoundTripper that
// saves events into the given WritableDB.
func (mx *Measurer) WrapDNSXRoundTripper(db WritableDB, rtx dnsx.RoundTripper) DNSXRoundTripper {
	return &dnsxRoundTripperDB{db: db, RoundTripper: rtx, begin: mx.Begin}
}

type dnsxRoundTripperDB struct {
	dnsx.RoundTripper
	begin time.Time
	db    WritableDB
}

// DNSRoundTripEvent contains the result of a DNS round trip.
type DNSRoundTripEvent struct {
	Network  string
	Address  string
	Query    []byte
	Started  float64
	Finished float64
	Error    error
	Reply    []byte
}

// MarshalJSON marshals a DNSRoundTripEvent to the archival
// format that is similar to df-002-dnst.
func (ev *DNSRoundTripEvent) MarshalJSON() ([]byte, error) {
	archival := NewArchivalDNSRoundTrip(ev)
	return json.Marshal(archival)
}

func (txp *dnsxRoundTripperDB) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Since(txp.begin).Seconds()
	reply, err := txp.RoundTripper.RoundTrip(ctx, query)
	finished := time.Since(txp.begin).Seconds()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		Network:  txp.RoundTripper.Network(),
		Address:  txp.RoundTripper.Address(),
		Query:    query,
		Started:  started,
		Finished: finished,
		Error:    err,
		Reply:    reply,
	})
	return reply, err
}
