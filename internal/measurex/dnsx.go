package measurex

//
// DNSX (DNS eXtensions)
//
// We wrap dnsx.RoundTripper to store events into a WritableDB.
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DNSXRoundTripper is a transport for sending raw DNS queries
// and receiving raw DNS replies. The internal/netxlite/dnsx
// package implements a bunch of these transports.
type DNSTransport = netxlite.DNSTransport

// WrapDNSXRoundTripper creates a new DNSXRoundTripper that
// saves events into the given WritableDB.
func (mx *Measurer) WrapDNSXRoundTripper(db WritableDB, rtx netxlite.DNSTransport) DNSTransport {
	return &dnsxRoundTripperDB{db: db, DNSTransport: rtx, begin: mx.Begin}
}

type dnsxRoundTripperDB struct {
	netxlite.DNSTransport
	begin time.Time
	db    WritableDB
}

// DNSRoundTripEvent contains the result of a DNS round trip.
type DNSRoundTripEvent struct {
	// This data structure is not in df-002-dns but the names and
	// semantics try to be consistent with such a spec.
	Network  string              `json:"engine"`
	Address  string              `json:"resolver_address"`
	Query    *ArchivalBinaryData `json:"raw_query"`
	Started  float64             `json:"started"`
	Finished float64             `json:"t"`
	Failure  *string             `json:"failure"`
	Reply    *ArchivalBinaryData `json:"raw_reply"`
}

func (txp *dnsxRoundTripperDB) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Since(txp.begin).Seconds()
	reply, err := txp.DNSTransport.RoundTrip(ctx, query)
	finished := time.Since(txp.begin).Seconds()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		Network:  txp.DNSTransport.Network(),
		Address:  txp.DNSTransport.Address(),
		Query:    NewArchivalBinaryData(query),
		Started:  started,
		Finished: finished,
		Failure:  NewArchivalFailure(err),
		Reply:    NewArchivalBinaryData(reply),
	})
	return reply, err
}
