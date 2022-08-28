package measurex

//
// DNSX (DNS eXtensions)
//
// We wrap dnsx.RoundTripper to store events into a WritableDB.
//

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// WrapDNSXRoundTripper creates a new DNSXRoundTripper that
// saves events into the given WritableDB.
func (mx *Measurer) WrapDNSXRoundTripper(db WritableDB, rtx model.DNSTransport) model.DNSTransport {
	return &dnsxRoundTripperDB{db: db, DNSTransport: rtx, begin: mx.Begin}
}

type dnsxRoundTripperDB struct {
	model.DNSTransport
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

func (txp *dnsxRoundTripperDB) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	started := time.Since(txp.begin).Seconds()
	response, err := txp.DNSTransport.RoundTrip(ctx, query)
	finished := time.Since(txp.begin).Seconds()
	txp.db.InsertIntoDNSRoundTrip(&DNSRoundTripEvent{
		Network:  tracex.ResolverNetworkAdaptNames(txp.DNSTransport.Network()),
		Address:  txp.DNSTransport.Address(),
		Query:    txp.maybeQueryBytes(query),
		Started:  started,
		Finished: finished,
		Failure:  NewFailure(err),
		Reply:    txp.maybeResponseBytes(response),
	})
	return response, err
}

func (txp *dnsxRoundTripperDB) maybeQueryBytes(query model.DNSQuery) []byte {
	data, _ := query.Bytes()
	return data
}

func (txp *dnsxRoundTripperDB) maybeResponseBytes(response model.DNSResponse) []byte {
	if response == nil {
		return nil
	}
	return response.Bytes()
}
