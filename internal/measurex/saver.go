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

// Saver is an EventDB that saves events and allows to
// ask questions regarding the saved events.
type Saver struct {
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

	begin         time.Time
	connID        int64
	measurementID int64
	mu            sync.Mutex
}

var _ EventDB = &Saver{}

// NewSaver creates a new instance of Saver.
func NewSaver(begin time.Time) *Saver {
	return &Saver{begin: begin}
}

// ElapsedTime implements EventDB.ElapsedTime.
func (s *Saver) ElapsedTime() time.Duration {
	return time.Since(s.begin)
}

// DeleteAll deletes all the saved data.
func (s *Saver) DeleteAll() {
	s.mu.Lock()
	s.dialTable = nil
	s.readWriteTable = nil
	s.closeTable = nil
	s.tlsHandshakeTable = nil
	s.lookupHostTable = nil
	s.lookupHTTPSvcTable = nil
	s.dnsRoundTripTable = nil
	s.httpRoundTripTable = nil
	s.httpRedirectTable = nil
	s.quicHandshakeTable = nil
	s.mu.Unlock()
}

// InsertIntoDial implements EventDB.InsertIntoDial.
func (s *Saver) InsertIntoDial(ev *NetworkEvent) {
	s.mu.Lock()
	s.dialTable = append(s.dialTable, ev)
	s.mu.Unlock()
}

// SelectAllFromDial returns all dial events.
func (s *Saver) SelectAllFromDial() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.dialTable...)
	s.mu.Unlock()
	return
}

// InsertIntoReadWrite implements EventDB.InsertIntoReadWrite.
func (s *Saver) InsertIntoReadWrite(ev *NetworkEvent) {
	s.mu.Lock()
	s.readWriteTable = append(s.readWriteTable, ev)
	s.mu.Unlock()
}

// SelectAllFromReadWrite returns all I/O events.
func (s *Saver) SelectAllFromReadWrite() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.readWriteTable...)
	s.mu.Unlock()
	return
}

// InsertIntoClose implements EventDB.InsertIntoClose.
func (s *Saver) InsertIntoClose(ev *NetworkEvent) {
	s.mu.Lock()
	s.closeTable = append(s.closeTable, ev)
	s.mu.Unlock()
}

// SelectAllFromClose returns all close events.
func (s *Saver) SelectAllFromClose() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.closeTable...)
	s.mu.Unlock()
	return
}

// InsertIntoTLSHandshake implements EventDB.InsertIntoTLSHandshake.
func (s *Saver) InsertIntoTLSHandshake(ev *TLSHandshakeEvent) {
	s.mu.Lock()
	s.tlsHandshakeTable = append(s.tlsHandshakeTable, ev)
	s.mu.Unlock()
}

// SelectAllFromTLSHandshake returns all TLS handshake events.
func (s *Saver) SelectAllFromTLSHandshake() (out []*TLSHandshakeEvent) {
	s.mu.Lock()
	out = append(out, s.tlsHandshakeTable...)
	s.mu.Unlock()
	return
}

// InsertIntoLookupHost implements EventDB.InsertIntoLookupHost.
func (s *Saver) InsertIntoLookupHost(ev *LookupHostEvent) {
	s.mu.Lock()
	s.lookupHostTable = append(s.lookupHostTable, ev)
	s.mu.Unlock()
}

// SelectAllFromLookupHost returns all the lookup host events.
func (s *Saver) SelectAllFromLookupHost() (out []*LookupHostEvent) {
	s.mu.Lock()
	out = append(out, s.lookupHostTable...)
	s.mu.Unlock()
	return
}

// InsertIntoHTTPSSvc implements EventDB.InsertIntoHTTPSSvc
func (s *Saver) InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent) {
	s.mu.Lock()
	s.lookupHTTPSvcTable = append(s.lookupHTTPSvcTable, ev)
	s.mu.Unlock()
}

// SelectAllFromLookupHTTPSSvc returns all HTTPSSvc lookup events.
func (s *Saver) SelectAllFromLookupHTTPSSvc() (out []*LookupHTTPSSvcEvent) {
	s.mu.Lock()
	out = append(out, s.lookupHTTPSvcTable...)
	s.mu.Unlock()
	return
}

// InsertIntoDNSRoundTrip implements EventDB.InsertIntoDNSRoundTrip.
func (s *Saver) InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent) {
	s.mu.Lock()
	s.dnsRoundTripTable = append(s.dnsRoundTripTable, ev)
	s.mu.Unlock()
}

// SelectAllFromDNSRoundTrip returns all DNS round trip events.
func (s *Saver) SelectAllFromDNSRoundTrip() (out []*DNSRoundTripEvent) {
	s.mu.Lock()
	out = append(out, s.dnsRoundTripTable...)
	s.mu.Unlock()
	return
}

// InsertIntoHTTPRoundTrip implements EventDB.InsertIntoHTTPRoundTrip.
func (s *Saver) InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent) {
	s.mu.Lock()
	s.httpRoundTripTable = append(s.httpRoundTripTable, ev)
	s.mu.Unlock()
}

// SelectAllFromHTTPRoundTrip returns all HTTP round trip events.
func (s *Saver) SelectAllFromHTTPRoundTrip() (out []*HTTPRoundTripEvent) {
	s.mu.Lock()
	out = append(out, s.httpRoundTripTable...)
	s.mu.Unlock()
	return
}

// InsertIntoHTTPRedirect implements EventDB.InsertIntoHTTPRedirect.
func (s *Saver) InsertIntoHTTPRedirect(ev *HTTPRedirectEvent) {
	s.mu.Lock()
	s.httpRedirectTable = append(s.httpRedirectTable, ev)
	s.mu.Unlock()
}

// SelectAllFromHTTPRedirect returns all HTTP redirections.
func (s *Saver) SelectAllFromHTTPRedirect() (out []*HTTPRedirectEvent) {
	s.mu.Lock()
	out = append(out, s.httpRedirectTable...)
	s.mu.Unlock()
	return
}

// InsertIntoQUICHandshake implements EventDB.InsertIntoQUICHandshake.
func (s *Saver) InsertIntoQUICHandshake(ev *QUICHandshakeEvent) {
	s.mu.Lock()
	s.quicHandshakeTable = append(s.quicHandshakeTable, ev)
	s.mu.Unlock()
}

// SelectAllFromQUICHandshake returns all QUIC handshake events.
func (s *Saver) SelectAllFromQUICHandshake() (out []*QUICHandshakeEvent) {
	s.mu.Lock()
	out = append(out, s.quicHandshakeTable...)
	s.mu.Unlock()
	return
}

// NextConnID implements EventDB.NextConnID.
func (s *Saver) NextConnID() (out int64) {
	s.mu.Lock()
	s.connID++ // start from 1
	out = s.connID
	s.mu.Unlock()
	return
}

// MeasurementID implements EventDB.MeasurementID.
func (s *Saver) MeasurementID() (out int64) {
	s.mu.Lock()
	out = s.measurementID
	s.mu.Unlock()
	return
}

// NextMeasurement increments the internal MeasurementID and
// returns it, so that later you can reference the current measurement.
func (s *Saver) NextMeasurement() (out int64) {
	s.mu.Lock()
	s.measurementID++ // start from 1
	out = s.measurementID
	s.mu.Unlock()
	return
}

// SelectAllFromDialWithMeasurementID calls SelectAllFromConnect
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromDialWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range s.SelectAllFromDial() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromReadWriteWithMeasurementID calls SelectAllFromReadWrite and
// filters the result by MeasurementID.
func (s *Saver) SelectAllFromReadWriteWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range s.SelectAllFromReadWrite() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromCloseWithMeasurementID calls SelectAllFromClose
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromCloseWithMeasurementID(id int64) (out []*NetworkEvent) {
	for _, ev := range s.SelectAllFromClose() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromTLSHandshakeWithMeasurementID calls SelectAllFromTLSHandshake
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromTLSHandshakeWithMeasurementID(id int64) (out []*TLSHandshakeEvent) {
	for _, ev := range s.SelectAllFromTLSHandshake() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromQUICHandshakeWithMeasurementID calls SelectAllFromQUICSHandshake
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromQUICHandshakeWithMeasurementID(id int64) (out []*QUICHandshakeEvent) {
	for _, ev := range s.SelectAllFromQUICHandshake() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromLookupHostWithMeasurementID calls SelectAllFromLookupHost
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromLookupHostWithMeasurementID(id int64) (out []*LookupHostEvent) {
	for _, ev := range s.SelectAllFromLookupHost() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromLookupHTTPSSvcWithMeasurementID calls SelectAllFromHTTPSSvc
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromLookupHTTPSSvcWithMeasurementID(id int64) (out []*LookupHTTPSSvcEvent) {
	for _, ev := range s.SelectAllFromLookupHTTPSSvc() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromDNSRoundTripWithMeasurementID calls SelectAllFromDNSRoundTrip
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromDNSRoundTripWithMeasurementID(id int64) (out []*DNSRoundTripEvent) {
	for _, ev := range s.SelectAllFromDNSRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromHTTPRoundTripWithMeasurementID calls SelectAllFromHTTPRoundTrip
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromHTTPRoundTripWithMeasurementID(id int64) (out []*HTTPRoundTripEvent) {
	for _, ev := range s.SelectAllFromHTTPRoundTrip() {
		if id == ev.MeasurementID {
			out = append(out, ev)
		}
	}
	return
}

// SelectAllFromHTTPRedirectWithMeasurementID calls SelectAllFromHTTPRedirect
// and filters the result by MeasurementID.
func (s *Saver) SelectAllFromHTTPRedirectWithMeasurementID(id int64) (out []*HTTPRedirectEvent) {
	for _, ev := range s.SelectAllFromHTTPRedirect() {
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
func (s *Saver) SelectAllEndpointsForDomain(domain, port string) (out []*Endpoint) {
	out = append(out, s.selectAllTCPEndpoints(domain, port)...)
	out = append(out, s.selectAllQUICEndpoints(domain, port)...)
	out = s.deduplicateEndpoints(out)
	return
}

func (s *Saver) selectAllTCPEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range s.SelectAllFromLookupHost() {
		if domain != entry.Domain {
			continue
		}
		for _, addr := range entry.Addrs {
			if net.ParseIP(addr) == nil {
				continue // skip CNAME entries courtesy the WCTH
			}
			out = append(out, s.newEndpoint(addr, port, NetworkTCP))
		}
	}
	return
}

func (s *Saver) selectAllQUICEndpoints(domain, port string) (out []*Endpoint) {
	for _, entry := range s.SelectAllFromLookupHTTPSSvc() {
		if domain != entry.Domain {
			continue
		}
		if !s.supportsHTTP3(entry) {
			continue
		}
		addrs := append([]string{}, entry.IPv4...)
		for _, addr := range append(addrs, entry.IPv6...) {
			out = append(out, s.newEndpoint(addr, port, NetworkQUIC))
		}
	}
	return
}

func (s *Saver) deduplicateEndpoints(epnts []*Endpoint) (out []*Endpoint) {
	duplicates := make(map[string]*Endpoint)
	for _, epnt := range epnts {
		duplicates[epnt.String()] = epnt
	}
	for _, epnt := range duplicates {
		out = append(out, epnt)
	}
	return
}

func (s *Saver) newEndpoint(addr, port string, network EndpointNetwork) *Endpoint {
	return &Endpoint{Network: network, Address: net.JoinHostPort(addr, port)}
}

func (s *Saver) supportsHTTP3(entry *LookupHTTPSSvcEvent) bool {
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

// SelectAllHTTPEndpointsForDomainAndMeasurementID returns all the
// HTTPEndpoints matching a specific domain and MeasurementID.
//
// Arguments:
//
// - URL is the URL for which we want endpoints;
//
// Returns a list of endpoints or an error.
func (s *Saver) SelectAllHTTPEndpointsForDomain(URL *url.URL) ([]*HTTPEndpoint, error) {
	domain := URL.Hostname()
	port, err := PortFromURL(URL)
	if err != nil {
		return nil, err
	}
	epnts := s.SelectAllEndpointsForDomain(domain, port)
	var out []*HTTPEndpoint
	for _, epnt := range epnts {
		out = append(out, &HTTPEndpoint{
			Domain:  domain,
			Network: epnt.Network,
			Address: epnt.Address,
			SNI:     domain,
			ALPN:    s.alpnForHTTPEndpoint(epnt.Network),
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

func (s *Saver) alpnForHTTPEndpoint(network EndpointNetwork) []string {
	switch network {
	case NetworkQUIC:
		return []string{"h3"}
	case NetworkTCP:
		return []string{"h2", "http/1.1"}
	default:
		return nil
	}
}
