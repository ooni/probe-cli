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
	Network  string
	Address  string
	Query    []byte
	Started  float64
	Finished float64
	Failure  *string
	Reply    []byte
}

func (txp *dnsxRoundTripperDB) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	started := time.Since(txp.begin).Seconds()
	reply, err := txp.DNSTransport.RoundTrip(ctx, query)
	finished := time.Since(txp.begin).Seconds()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		Network:  txp.DNSTransport.Network(),
		Address:  txp.DNSTransport.Address(),
		Query:    query,
		Started:  started,
		Finished: finished,
		Failure:  NewFailure(err),
		Reply:    reply,
	})
	return reply, err
}
