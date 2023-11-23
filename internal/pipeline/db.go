package pipeline

import (
	"errors"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

// DB is a database containing observations.
//
// This struct is not goroutine safe. The zero value is invalid. Use the
// [NewDB] to construct a valid instance.
type DB struct {
	dnsByTxID       map[int64]*DNSObservation
	thDNSAddrs      map[string]bool
	thDNSFailure    *string
	thEpntByEpnt    map[string]*EndpointObservationTH
	thWeb           optional.Value[*WebObservationTH]
	urlHostname     string
	webByTxID       map[int64]*WebEndpointObservation
	webFinalRequest optional.Value[*WebEndpointObservation]
}

// NewObservationsDB constructs a new [*DB] instance.
func NewDB() *DB {
	return &DB{
		dnsByTxID:       map[int64]*DNSObservation{},
		thDNSAddrs:      map[string]bool{},
		thDNSFailure:    nil,
		thEpntByEpnt:    map[string]*EndpointObservationTH{},
		thWeb:           optional.None[*WebObservationTH](),
		urlHostname:     "",
		webByTxID:       map[int64]*WebEndpointObservation{},
		webFinalRequest: optional.None[*WebEndpointObservation](),
	}
}

// Ingest ingests measurement results and populates the database.
func (db *DB) Ingest(m *CanonicalMeasurement) error {
	// Extra the hostname from the input URL
	URL, err := url.Parse(m.Input)
	if err != nil {
		return err
	}
	db.urlHostname = URL.Hostname()

	// Obtain the test keys or stop early
	tk := m.TestKeys.UnwrapOr(nil)
	if tk == nil {
		return nil
	}

	// Build knowledge about existing TCP endpoints
	if err := db.addNetworkEventsTCPConnect(tk.NetworkEvents...); err != nil {
		return err
	}
	if err := db.addTLSHandshakeEvents(tk.TLSHandshakes...); err != nil {
		return err
	}

	// Build knowledge about QUIC endpoints
	if err := db.addQUICHandshakeEvents(tk.QUICHandshakes...); err != nil {
		return err
	}

	// Enrich dataset with HTTP round trips information
	if err := db.addHTTPRoundTrips(tk.Requests...); err != nil {
		return err
	}

	// Build knowledge about DNS lookups.
	if err := db.addDNSLookups(tk.Queries...); err != nil {
		return err
	}

	// Process a control response if available
	if thResp := tk.Control.UnwrapOr(nil); thResp != nil {
		// Add DNS results first
		if err := db.thAddDNS(thResp); err != nil {
			return err
		}

		// Then create TCP connect entries
		if err := db.thAddTCPConnect(thResp); err != nil {
			return err
		}

		// Then extend information using TLS
		if err := db.thAddTLSHandshake(thResp); err != nil {
			return err
		}

		// Finally, add information about HTTP
		if err := db.thAddHTTPResponse(thResp); err != nil {
			return err
		}
	}

	// Cross reference data structures.
	db.buildXrefsDNS()
	db.buildXrefTH()
	if err := db.maybeFindFinalRequest(); err != nil {
		return err
	}

	return nil
}

func (db *DB) buildXrefsDNS() {
	// map addresses to who resolved them
	addrToGetaddrinfo := make(map[string][]*DNSObservation)
	addrToUDP := make(map[string][]*DNSObservation)
	addrToHTTPS := make(map[string][]*DNSObservation)
	for _, dobs := range db.dnsByTxID {
		switch dobs.Engine {
		case "system", "getaddrinfo", "golang_net_resolver", "go":
			for _, addr := range dobs.IPAddrs {
				addrToGetaddrinfo[addr] = append(addrToGetaddrinfo[addr], dobs)
			}

		case "udp":
			for _, addr := range dobs.IPAddrs {
				addrToUDP[addr] = append(addrToUDP[addr], dobs)
			}

		case "doh":
			for _, addr := range dobs.IPAddrs {
				addrToHTTPS[addr] = append(addrToHTTPS[addr], dobs)
			}
		}
	}

	// create cross references inside the endpoints
	for _, wobs := range db.webByTxID {
		wobs.DNSLookupGetaddrinfoXref = addrToGetaddrinfo[wobs.IPAddress]
		wobs.DNSLookupUDPXref = addrToUDP[wobs.IPAddress]
		wobs.DNSLookupHTTPSXref = addrToHTTPS[wobs.IPAddress]
	}
}

func (db *DB) buildXrefTH() {
	for _, wobs := range db.webByTxID {
		// create cross references with TH DNS lookups
		_, ok := db.thDNSAddrs[wobs.IPAddress]
		wobs.DNSLookupTHXref = ok

		// create cross references with TH endpoints
		if xref, ok := db.thEpntByEpnt[wobs.Endpoint]; ok {
			wobs.THEndpointXref = optional.Some(xref)
		}
	}
}

var errMultipleFinalRequests = errors.New("analysis: multiple final requests")

func (db *DB) maybeFindFinalRequest() error {
	// find all the possible final request candidates
	var finals []*WebEndpointObservation
	for _, wobs := range db.webByTxID {
		switch code := wobs.HTTPResponseStatusCode.UnwrapOr(0); code {
		case 0, 301, 302, 307, 308:
			// this is a redirect or a nonexisting response in the case of zero

		default:
			// found candidate
			finals = append(finals, wobs)
		}
	}

	// Implementation note: the final request is a request that is not a redirect and
	// we expect to see just one of them. This code is written assuming we will have
	// more than a final request in the future and to fail in such a case.
	switch {
	case len(finals) > 1:
		return errMultipleFinalRequests

	case len(finals) == 1:
		db.webFinalRequest = optional.Some(finals[0])
		return nil

	default:
		return nil
	}
}
