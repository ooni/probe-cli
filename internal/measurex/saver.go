package measurex

import "sync"

// Saver is a DB that saves measurements.
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
	connID             int64
	measurementID      int64
	mu                 sync.Mutex
}

func (s *Saver) InsertIntoDial(ev *NetworkEvent) {
	s.mu.Lock()
	s.dialTable = append(s.dialTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromDial() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.dialTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoReadWrite(ev *NetworkEvent) {
	s.mu.Lock()
	s.readWriteTable = append(s.readWriteTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromReadWrite() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.readWriteTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoClose(ev *NetworkEvent) {
	s.mu.Lock()
	s.closeTable = append(s.closeTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromClose() (out []*NetworkEvent) {
	s.mu.Lock()
	out = append(out, s.closeTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoTLSHandshake(ev *TLSHandshakeEvent) {
	s.mu.Lock()
	s.tlsHandshakeTable = append(s.tlsHandshakeTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromTLSHandshake() (out []*TLSHandshakeEvent) {
	s.mu.Lock()
	out = append(out, s.tlsHandshakeTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoLookupHost(ev *LookupHostEvent) {
	s.mu.Lock()
	s.lookupHostTable = append(s.lookupHostTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromLookupHost() (out []*LookupHostEvent) {
	s.mu.Lock()
	out = append(out, s.lookupHostTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent) {
	s.mu.Lock()
	s.lookupHTTPSvcTable = append(s.lookupHTTPSvcTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromLookupHTTPSSvc() (out []*LookupHTTPSSvcEvent) {
	s.mu.Lock()
	out = append(out, s.lookupHTTPSvcTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent) {
	s.mu.Lock()
	s.dnsRoundTripTable = append(s.dnsRoundTripTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromDNSRoundTrip() (out []*DNSRoundTripEvent) {
	s.mu.Lock()
	out = append(out, s.dnsRoundTripTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent) {
	s.mu.Lock()
	s.httpRoundTripTable = append(s.httpRoundTripTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromHTTPRoundTrip() (out []*HTTPRoundTripEvent) {
	s.mu.Lock()
	out = append(out, s.httpRoundTripTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) InsertIntoHTTPRedirect(ev *HTTPRedirectEvent) {
	s.mu.Lock()
	s.httpRedirectTable = append(s.httpRedirectTable, ev)
	s.mu.Unlock()
}

func (s *Saver) SelectAllFromHTTPRedirect() (out []*HTTPRedirectEvent) {
	s.mu.Lock()
	out = append(out, s.httpRedirectTable...)
	s.mu.Unlock()
	return
}

func (s *Saver) NextConnID() (out int64) {
	s.mu.Lock()
	s.connID++ // start from 1
	out = s.connID
	s.mu.Unlock()
	return
}

func (s *Saver) MeasurementID() (out int64) {
	s.mu.Lock()
	out = s.measurementID
	s.mu.Unlock()
	return
}

func (s *Saver) NextMeasurement() (out int64) {
	s.mu.Lock()
	s.measurementID++ // start from 1
	out = s.measurementID
	s.mu.Unlock()
	return
}
