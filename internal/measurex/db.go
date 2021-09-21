package measurex

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// EventDB is a "database" holding events records as seen by the
// networking code that needs to save events.
type EventDB interface {
	// ElapsedTime returns the elapsed time since the beginning
	// of time as configured into the database.
	ElapsedTime() time.Duration

	// InsertIntoDial saves a Dial event.
	InsertIntoDial(ev *NetworkEvent)

	// InsertIntoReadWrite saves an I/O event.
	InsertIntoReadWrite(ev *NetworkEvent)

	// InsertIntoClose saves a close event.
	InsertIntoClose(ev *NetworkEvent)

	// InsertIntoTLSHandshake saves a TLS handshake event.
	InsertIntoTLSHandshake(ev *TLSHandshakeEvent)

	// InsertIntoLookupHost saves a lookup host event.
	InsertIntoLookupHost(ev *LookupHostEvent)

	// InsertIntoLookupHTTPSvc saves an HTTPSvc lookup event.
	InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent)

	// InsertIntoDNSRoundTrip saves a DNS round trip event.
	InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent)

	// InsertIntoHTTPRoundTrip saves an HTTP round trip event.
	InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent)

	// InsertIntoHTTPRedirect saves an HTTP redirect event.
	InsertIntoHTTPRedirect(ev *HTTPRedirectEvent)

	// InsertIntoQUICHandshake saves a QUIC handshake event.
	InsertIntoQUICHandshake(ev *QUICHandshakeEvent)

	// NextConnID increments and returns the connection ID.
	NextConnID() int64

	// MeasurementID returns the current measurement ID.
	MeasurementID() int64
}

// DB is an EventDB that saves events and also allows to
// ask questions regarding the saved events.
//
// Caveats: the way in which we assign an ID to a measurement
// (i.e., a related set of operations) assumes that we don't
// run paralell measurements with the same DB. To run parallel
// measurements, you need to create a children DB using the
// NewChildrenDB factory function.
type DB struct {
	*dbSharedWithChildren

	// the measurementID must not be shared with children
	// running in parallel because it must stay const during
	// the whole measurements or we cannot measure.
	//
	// Use NewChildrenDB to create one or more children
	// that can run in parallel with each other and their parent.
	measurementID int64
	mu            sync.Mutex
}

// dbSharedWithChildren is the part of the database shared
// with children databases. These databases are created during
// parallel measurements but need to insert into the same
// tables of the parent database. The ConnID can also be part
// of this structure. The only element that needs to be constant
// during a measurement is the MeasurementID.
type dbSharedWithChildren struct {
	// database tables
	dialTable          []*NetworkEvent
	readWriteTable     []*NetworkEvent
	closeTable         []*NetworkEvent
	tlsHandshakeTable  []*TLSHandshakeEvent
	lookupHostTable    []*LookupHostEvent
	lookupHTTPSvcTable []*LookupHTTPSSvcEvent
	dnsRoundTripTable  []*DNSRoundTripEvent
	httpRoundTripTable []*HTTPRoundTripEvent
	httpRedirectTable  []*HTTPRedirectEvent
	quicHandshakeTable []*QUICHandshakeEvent
	resolversTable     []*ResolverInfo
	testHelpersTable   []*TestHelperInfo

	// mu protects all the above tables and ConnID
	mu sync.Mutex

	// other pieces of data we can share with children
	// because they have more relaxed constaints than the
	// measurement ID: begin is read-only and ConnID is
	// read once and then propagated.
	begin  time.Time
	connID int64
}

var _ EventDB = &DB{}

var (
	baseMeasurementID int32
	measurementIDmu   sync.Mutex
)

func nextBaseMeasurementID() (out int64) {
	measurementIDmu.Lock()
	baseMeasurementID++
	out = (int64(baseMeasurementID) << 32)
	measurementIDmu.Unlock()
	return
}

// NewDB creates a new instance of DB. This instance will have the base
// time configured to be begin. Also, its base measurement ID will depend
// on how many databases we have created so far. Each database gets its
// own 31 bit namespace for measurements.
func NewDB(begin time.Time) *DB {
	return &DB{
		dbSharedWithChildren: &dbSharedWithChildren{begin: begin},
		measurementID:        nextBaseMeasurementID(),
	}
}

// NewChildDB returns a DB that has the same base measurement and
// tables of the original DB but gets a new measurement namespace. You
// should use this factory every time you run parallel measurements.
func (db *DB) NewChildDB() *DB {
	return &DB{
		dbSharedWithChildren: db.dbSharedWithChildren,
		measurementID:        nextBaseMeasurementID(),
	}
}

// ElapsedTime implements EventDB.ElapsedTime.
func (db *dbSharedWithChildren) ElapsedTime() time.Duration {
	return time.Since(db.begin)
}

// DeleteAll deletes all the saved data.
func (db *dbSharedWithChildren) DeleteAll() {
	db.mu.Lock()
	db.dialTable = nil
	db.readWriteTable = nil
	db.closeTable = nil
	db.tlsHandshakeTable = nil
	db.lookupHostTable = nil
	db.lookupHTTPSvcTable = nil
	db.dnsRoundTripTable = nil
	db.httpRoundTripTable = nil
	db.httpRedirectTable = nil
	db.quicHandshakeTable = nil
	db.mu.Unlock()
}

// InsertIntoDial implements EventDB.InsertIntoDial.
func (db *dbSharedWithChildren) InsertIntoDial(ev *NetworkEvent) {
	db.mu.Lock()
	db.dialTable = append(db.dialTable, ev)
	db.mu.Unlock()
}

// SelectAllFromDial returns all dial events.
func (db *dbSharedWithChildren) SelectAllFromDial() (out []*NetworkEvent) {
	db.mu.Lock()
	out = append(out, db.dialTable...)
	db.mu.Unlock()
	return
}

// InsertIntoReadWrite implements EventDB.InsertIntoReadWrite.
func (db *dbSharedWithChildren) InsertIntoReadWrite(ev *NetworkEvent) {
	db.mu.Lock()
	db.readWriteTable = append(db.readWriteTable, ev)
	db.mu.Unlock()
}

// SelectAllFromReadWrite returns all I/O events.
func (db *dbSharedWithChildren) SelectAllFromReadWrite() (out []*NetworkEvent) {
	db.mu.Lock()
	out = append(out, db.readWriteTable...)
	db.mu.Unlock()
	return
}

// InsertIntoClose implements EventDB.InsertIntoClose.
func (db *dbSharedWithChildren) InsertIntoClose(ev *NetworkEvent) {
	db.mu.Lock()
	db.closeTable = append(db.closeTable, ev)
	db.mu.Unlock()
}

// SelectAllFromClose returns all close events.
func (db *dbSharedWithChildren) SelectAllFromClose() (out []*NetworkEvent) {
	db.mu.Lock()
	out = append(out, db.closeTable...)
	db.mu.Unlock()
	return
}

// InsertIntoTLSHandshake implements EventDB.InsertIntoTLSHandshake.
func (db *dbSharedWithChildren) InsertIntoTLSHandshake(ev *TLSHandshakeEvent) {
	db.mu.Lock()
	db.tlsHandshakeTable = append(db.tlsHandshakeTable, ev)
	db.mu.Unlock()
}

// SelectAllFromTLSHandshake returns all TLS handshake events.
func (db *dbSharedWithChildren) SelectAllFromTLSHandshake() (out []*TLSHandshakeEvent) {
	db.mu.Lock()
	out = append(out, db.tlsHandshakeTable...)
	db.mu.Unlock()
	return
}

// InsertIntoLookupHost implements EventDB.InsertIntoLookupHost.
func (db *dbSharedWithChildren) InsertIntoLookupHost(ev *LookupHostEvent) {
	db.mu.Lock()
	db.lookupHostTable = append(db.lookupHostTable, ev)
	db.mu.Unlock()
}

// SelectAllFromLookupHost returns all the lookup host events.
func (db *dbSharedWithChildren) SelectAllFromLookupHost() (out []*LookupHostEvent) {
	db.mu.Lock()
	out = append(out, db.lookupHostTable...)
	db.mu.Unlock()
	return
}

// InsertIntoHTTPSSvc implements EventDB.InsertIntoHTTPSSvc
func (db *dbSharedWithChildren) InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent) {
	db.mu.Lock()
	db.lookupHTTPSvcTable = append(db.lookupHTTPSvcTable, ev)
	db.mu.Unlock()
}

// SelectAllFromLookupHTTPSSvc returns all HTTPSSvc lookup events.
func (db *dbSharedWithChildren) SelectAllFromLookupHTTPSSvc() (out []*LookupHTTPSSvcEvent) {
	db.mu.Lock()
	out = append(out, db.lookupHTTPSvcTable...)
	db.mu.Unlock()
	return
}

// InsertIntoDNSRoundTrip implements EventDB.InsertIntoDNSRoundTrip.
func (db *dbSharedWithChildren) InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent) {
	db.mu.Lock()
	db.dnsRoundTripTable = append(db.dnsRoundTripTable, ev)
	db.mu.Unlock()
}

// SelectAllFromDNSRoundTrip returns all DNS round trip events.
func (db *dbSharedWithChildren) SelectAllFromDNSRoundTrip() (out []*DNSRoundTripEvent) {
	db.mu.Lock()
	out = append(out, db.dnsRoundTripTable...)
	db.mu.Unlock()
	return
}

// InsertIntoHTTPRoundTrip implements EventDB.InsertIntoHTTPRoundTrip.
func (db *dbSharedWithChildren) InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent) {
	db.mu.Lock()
	db.httpRoundTripTable = append(db.httpRoundTripTable, ev)
	db.mu.Unlock()
}

// SelectAllFromHTTPRoundTrip returns all HTTP round trip events.
func (db *dbSharedWithChildren) SelectAllFromHTTPRoundTrip() (out []*HTTPRoundTripEvent) {
	db.mu.Lock()
	out = append(out, db.httpRoundTripTable...)
	db.mu.Unlock()
	return
}

// InsertIntoHTTPRedirect implements EventDB.InsertIntoHTTPRedirect.
func (db *dbSharedWithChildren) InsertIntoHTTPRedirect(ev *HTTPRedirectEvent) {
	db.mu.Lock()
	db.httpRedirectTable = append(db.httpRedirectTable, ev)
	db.mu.Unlock()
}

// SelectAllFromHTTPRedirect returns all HTTP redirections.
func (db *dbSharedWithChildren) SelectAllFromHTTPRedirect() (out []*HTTPRedirectEvent) {
	db.mu.Lock()
	out = append(out, db.httpRedirectTable...)
	db.mu.Unlock()
	return
}

// InsertIntoQUICHandshake implements EventDB.InsertIntoQUICHandshake.
func (db *dbSharedWithChildren) InsertIntoQUICHandshake(ev *QUICHandshakeEvent) {
	db.mu.Lock()
	db.quicHandshakeTable = append(db.quicHandshakeTable, ev)
	db.mu.Unlock()
}

// SelectAllFromQUICHandshake returns all QUIC handshake events.
func (db *dbSharedWithChildren) SelectAllFromQUICHandshake() (out []*QUICHandshakeEvent) {
	db.mu.Lock()
	out = append(out, db.quicHandshakeTable...)
	db.mu.Unlock()
	return
}

// ResolverInfo contains info about a DNS resolver.
type ResolverInfo struct {
	// Network is the resolver's network (e.g., "doh", "udp")
	Network string

	// Address is the address (e.g., "1.1.1.1:53", "https://1.1.1.1/dns-query")
	Address string
}

// string returns a string representation of the resolver.
func (ri *ResolverInfo) string() string {
	return fmt.Sprintf("%s@%s", ri.Network, ri.Address)
}

// InsertIntoResolvers inserts a given resolver into the resolver's table.
func (db *dbSharedWithChildren) InsertIntoResolvers(network, address string) {
	db.mu.Lock()
	db.resolversTable = append(db.resolversTable, &ResolverInfo{
		Network: network,
		Address: address,
	})
	db.mu.Unlock()
}

// SelectAllFromResolvers returns all the configured resolvers. This function
// ensures that the system resolver is in the list and also ensures that we
// return in output a list only containing unique resolvers.
func (db *dbSharedWithChildren) SelectAllFromResolvers() (out []*ResolverInfo) {
	all := append([]*ResolverInfo{}, &ResolverInfo{Network: "system"})
	db.mu.Lock()
	all = append(all, db.resolversTable...)
	db.mu.Unlock()
	unique := make(map[string]bool)
	for _, reso := range all {
		if _, found := unique[reso.string()]; found {
			continue
		}
		unique[reso.string()] = true
		out = append(out, reso)
	}
	return
}

// TestHelperInfo contains info about a test helper.
type TestHelperInfo struct {
	// Protocol is the test helpers's protocol (e.g., "wcth")
	Protocol string

	// URL is the URL (e.g., "https://wcth.ooni.io/")
	URL string
}

// string returns a string representation of the resolver.
func (ti *TestHelperInfo) string() string {
	return fmt.Sprintf("%s@%s", ti.Protocol, ti.URL)
}

// InsertIntoTestHelpers inserts a given TH into the test helpers's table.
func (db *dbSharedWithChildren) InsertIntoTestHelpers(proto, URL string) {
	db.mu.Lock()
	db.testHelpersTable = append(db.testHelpersTable, &TestHelperInfo{
		Protocol: proto,
		URL:      URL,
	})
	db.mu.Unlock()
}

// SelectAllFromTestHelperss returns all the configured THs. This function
// ensures that we return in output a list only containing unique THs.
func (db *dbSharedWithChildren) SelectAllFromTestHelpers() (out []*TestHelperInfo) {
	var all []*TestHelperInfo
	db.mu.Lock()
	all = append(all, db.testHelpersTable...)
	db.mu.Unlock()
	unique := make(map[string]bool)
	for _, th := range all {
		if _, found := unique[th.string()]; found {
			continue
		}
		unique[th.string()] = true
		out = append(out, th)
	}
	return
}

// NextConnID implements EventDB.NextConnID.
func (db *dbSharedWithChildren) NextConnID() (out int64) {
	db.mu.Lock()
	db.connID++ // start from 1
	out = db.connID
	db.mu.Unlock()
	return
}

// MeasurementID implements EventDB.MeasurementID.
func (db *DB) MeasurementID() (out int64) {
	db.mu.Lock()
	out = db.measurementID
	db.mu.Unlock()
	return
}

// NextMeasurement increments the internal MeasurementID and
// returns it, so that later you can reference the current measurement.
func (db *DB) NextMeasurement() (out int64) {
	db.mu.Lock()
	db.measurementID++ // start from 1
	out = db.measurementID
	db.mu.Unlock()
	return
}

// SelectAllFromDialWithMeasurementID calls SelectAllFromConnect
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromDialWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range db.SelectAllFromDial() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromReadWriteWithMeasurementID calls SelectAllFromReadWrite and
// filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromReadWriteWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range db.SelectAllFromReadWrite() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromCloseWithMeasurementID calls SelectAllFromClose
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromCloseWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range db.SelectAllFromClose() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromTLSHandshakeWithMeasurementID calls SelectAllFromTLSHandshake
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromTLSHandshakeWithMeasurementID(id int64) (out []*TLSHandshakeEvent) {
	for _, ev := range db.SelectAllFromTLSHandshake() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromQUICHandshakeWithMeasurementID calls SelectAllFromQUICSHandshake
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromQUICHandshakeWithMeasurementID(id int64) (out []*QUICHandshakeEvent) {
	for _, ev := range db.SelectAllFromQUICHandshake() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromLookupHostWithMeasurementID calls SelectAllFromLookupHost
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromLookupHostWithMeasurementID(id int64) (out []*LookupHostEvent) {
	for _, ev := range db.SelectAllFromLookupHost() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromLookupHTTPSSvcWithMeasurementID calls SelectAllFromHTTPSSvc
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromLookupHTTPSSvcWithMeasurementID(id int64) (out []*LookupHTTPSSvcEvent) {
	for _, ev := range db.SelectAllFromLookupHTTPSSvc() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromDNSRoundTripWithMeasurementID calls SelectAllFromDNSRoundTrip
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromDNSRoundTripWithMeasurementID(id int64) (out []*DNSRoundTripEvent) {
	for _, ev := range db.SelectAllFromDNSRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromHTTPRoundTripWithMeasurementID calls SelectAllFromHTTPRoundTrip
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromHTTPRoundTripWithMeasurementID(id int64) (out []*HTTPRoundTripEvent) {
	for _, ev := range db.SelectAllFromHTTPRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromHTTPRedirectWithMeasurementID calls SelectAllFromHTTPRedirect
// and filters the result by MeasurementID.
func (db *dbSharedWithChildren) SelectAllFromHTTPRedirectWithMeasurementID(id int64) (out []*HTTPRedirectEvent) {
	for _, ev := range db.SelectAllFromHTTPRedirect() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// EndpointNetwork is the network of an endpoint.
type EndpointNetwork string

const (
	// NetworkTCP identifies endpoints using TCP.
	NetworkTCP = EndpointNetwork("tcp")

	// NetworkQUIC identifies endpoints using QUIC.
	NetworkQUIC = EndpointNetwork("quic")
)

// Endpoint is an endpoint for a domain.
type Endpoint struct {
	// Network is the network (e.g., "tcp", "quic")
	Network EndpointNetwork

	// Address is the endpoint address (e.g., "8.8.8.8:443")
	Address string
}

// String converts an endpoint to a string (e.g., "8.8.8.8:443/tcp")
func (e *Endpoint) String() string {
	return fmt.Sprintf("%s/%s", e.Address, e.Network)
}

// SelectAllEndpointsForDomain returns all the
// endpoints for a specific domain.
//
// Arguments:
//
// - domain is the domain we want to connect to;
//
// - port is the port for the endpoint.
func (db *dbSharedWithChildren) SelectAllEndpointsForDomain(domain, port string) (out []*Endpoint) {
	out = append(out, db.selectAllTCPEndpoints(domain, port)...)
	out = append(out, db.selectAllQUICEndpoints(domain, port)...)
	out = db.deduplicateEndpoints(out)
	return
}

func (db *dbSharedWithChildren) selectAllTCPEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range db.SelectAllFromLookupHost() {
		if domain != entry.Domain {
			continue
		}
		for _, addr := range entry.Addrs {
			if net.ParseIP(addr) == nil {
				continue // skip CNAME entries courtesy the WCTH
			}
			out = append(out, db.newEndpoint(addr, port, NetworkTCP))
		}
	}
	return
}

func (db *dbSharedWithChildren) selectAllQUICEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range db.SelectAllFromLookupHTTPSSvc() {
		if domain != entry.Domain {
			continue
		}
		if !db.supportsHTTP3(entry) {
			continue
		}
		addrs := append([]string{}, entry.IPv4...)
		for _, addr := range append(addrs, entry.IPv6...) {
			out = append(out, db.newEndpoint(addr, port, NetworkQUIC))
		}
	}
	return
}

func (db *dbSharedWithChildren) deduplicateEndpoints(epnts []*Endpoint) (out []*Endpoint) {
	duplicates := make(map[string]*Endpoint)
	for _, epnt := range epnts {
		duplicates[epnt.String()] = epnt
	}
	for _, epnt := range duplicates {
		out = append(out, epnt)
	}
	return
}

func (db *dbSharedWithChildren) newEndpoint(addr, port string, network EndpointNetwork) *Endpoint {
	return &Endpoint{Network: network, Address: net.JoinHostPort(addr, port)}
}

func (db *dbSharedWithChildren) supportsHTTP3(entry *LookupHTTPSSvcEvent) bool {
	for _, alpn := range entry.ALPN {
		switch alpn {
		case "h3":
			return true
		}
	}
	return false
}

// HTTPEndpoint is an HTTP/HTTPS/HTTP3 endpoint.
type HTTPEndpoint struct {
	// Domain is the endpoint domain (e.g., "dns.google").
	Domain string

	// Network is the network (e.g., "tcp" or "quic").
	Network EndpointNetwork

	// Address is the endpoint address (e.g., "8.8.8.8:443").
	Address string

	// SNI is the SNI to use (only used with URL.scheme == "https").
	SNI string

	// ALPN is the ALPN to use (only used with URL.scheme == "https").
	ALPN []string

	// URL is the endpoint URL.
	URL *url.URL

	// Header contains request headers.
	Header http.Header
}

// String converts an HTTP endpoint to a string (e.g., "8.8.8.8:443/tcp")
func (e *HTTPEndpoint) String() string {
	return fmt.Sprintf("%s/%s", e.Address, e.Network)
}

// SelectAllHTTPEndpointsForURL returns all the
// HTTPEndpoints matching a specific URL' domain.
//
// Arguments:
//
// - URL is the URL for which we want endpoints;
//
// Returns a list of endpoints or an error.
func (db *dbSharedWithChildren) SelectAllHTTPEndpointsForURL(URL *url.URL) ([]*HTTPEndpoint, error) {
	domain := URL.Hostname()
	port, err := PortFromURL(URL)
	if err != nil {
		return nil, err
	}
	epnts := db.SelectAllEndpointsForDomain(domain, port)
	var out []*HTTPEndpoint
	for _, epnt := range epnts {
		if URL.Scheme != "https" && epnt.Network == NetworkQUIC {
			continue // we'll only use QUIC with HTTPS
		}
		out = append(out, &HTTPEndpoint{
			Domain:  domain,
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     domain,
			ALPN:    db.alpnForHTTPEndpoint(epnt.Network),
			URL:     URL,
			Header:  NewHTTPRequestHeaderForMeasuring(),
		})
	}
	return out, nil
}

// ErrCannotDeterminePortFromURL indicates that we could not determine
// the correct port from the URL authority and scheme.
var ErrCannotDeterminePortFromURL = errors.New("cannot determine port from URL")

// PortFromURL returns the port determined from the URL or an error.
func PortFromURL(URL *url.URL) (string, error) {
	switch {
	case URL.Port() != "":
		return URL.Port(), nil
	case URL.Scheme == "https":
		return "443", nil
	case URL.Scheme == "http":
		return "80", nil
	default:
		return "", ErrCannotDeterminePortFromURL
	}
}

func (db *dbSharedWithChildren) alpnForHTTPEndpoint(network EndpointNetwork) []string {
	switch network {
	case NetworkQUIC:
		return []string{"h3"}
	case NetworkTCP:
		return []string{"h2", "http/1.1"}
	default:
		return nil
	}
}
