package measurex

//
// DB
//
// This file defines two types:
//
// - WritableDB is the interface for storing events that
// we pass to the networking code
//
// - MeasurementDB is a concrete database in which network
// code stores events and from which you can create a
// measurement with all the collected events
//

import (
	"sync"
)

// WritableDB is a measurement database in which you can write.
type WritableDB interface {
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
}

// MeasurementDB is a database for assembling a measurement.
type MeasurementDB struct {
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

	// mu protects all the fields
	mu sync.Mutex
}

var _ WritableDB = &MeasurementDB{}

// InsertIntoDial implements EventDB.InsertIntoDial.
func (db *MeasurementDB) InsertIntoDial(ev *NetworkEvent) {
	db.mu.Lock()
	db.dialTable = append(db.dialTable, ev)
	db.mu.Unlock()
}

// selectAllFromDial returns all dial events.
func (db *MeasurementDB) selectAllFromDial() (out []*NetworkEvent) {
	out = append(out, db.dialTable...)
	return
}

// InsertIntoReadWrite implements EventDB.InsertIntoReadWrite.
func (db *MeasurementDB) InsertIntoReadWrite(ev *NetworkEvent) {
	db.mu.Lock()
	db.readWriteTable = append(db.readWriteTable, ev)
	db.mu.Unlock()
}

// selectAllFromReadWrite returns all I/O events.
func (db *MeasurementDB) selectAllFromReadWrite() (out []*NetworkEvent) {
	out = append(out, db.readWriteTable...)
	return
}

// InsertIntoClose implements EventDB.InsertIntoClose.
func (db *MeasurementDB) InsertIntoClose(ev *NetworkEvent) {
	db.mu.Lock()
	db.closeTable = append(db.closeTable, ev)
	db.mu.Unlock()
}

// selectAllFromClose returns all close events.
func (db *MeasurementDB) selectAllFromClose() (out []*NetworkEvent) {
	out = append(out, db.closeTable...)
	return
}

// InsertIntoTLSHandshake implements EventDB.InsertIntoTLSHandshake.
func (db *MeasurementDB) InsertIntoTLSHandshake(ev *TLSHandshakeEvent) {
	db.mu.Lock()
	db.tlsHandshakeTable = append(db.tlsHandshakeTable, ev)
	db.mu.Unlock()
}

// selectAllFromTLSHandshake returns all TLS handshake events.
func (db *MeasurementDB) selectAllFromTLSHandshake() (out []*TLSHandshakeEvent) {
	out = append(out, db.tlsHandshakeTable...)
	return
}

// InsertIntoLookupHost implements EventDB.InsertIntoLookupHost.
func (db *MeasurementDB) InsertIntoLookupHost(ev *LookupHostEvent) {
	db.mu.Lock()
	db.lookupHostTable = append(db.lookupHostTable, ev)
	db.mu.Unlock()
}

// selectAllFromLookupHost returns all the lookup host events.
func (db *MeasurementDB) selectAllFromLookupHost() (out []*LookupHostEvent) {
	out = append(out, db.lookupHostTable...)
	return
}

// InsertIntoHTTPSSvc implements EventDB.InsertIntoHTTPSSvc
func (db *MeasurementDB) InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent) {
	db.mu.Lock()
	db.lookupHTTPSvcTable = append(db.lookupHTTPSvcTable, ev)
	db.mu.Unlock()
}

// selectAllFromLookupHTTPSSvc returns all HTTPSSvc lookup events.
func (db *MeasurementDB) selectAllFromLookupHTTPSSvc() (out []*LookupHTTPSSvcEvent) {
	out = append(out, db.lookupHTTPSvcTable...)
	return
}

// InsertIntoDNSRoundTrip implements EventDB.InsertIntoDNSRoundTrip.
func (db *MeasurementDB) InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent) {
	db.mu.Lock()
	db.dnsRoundTripTable = append(db.dnsRoundTripTable, ev)
	db.mu.Unlock()
}

// selectAllFromDNSRoundTrip returns all DNS round trip events.
func (db *MeasurementDB) selectAllFromDNSRoundTrip() (out []*DNSRoundTripEvent) {
	out = append(out, db.dnsRoundTripTable...)
	return
}

// InsertIntoHTTPRoundTrip implements EventDB.InsertIntoHTTPRoundTrip.
func (db *MeasurementDB) InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent) {
	db.mu.Lock()
	db.httpRoundTripTable = append(db.httpRoundTripTable, ev)
	db.mu.Unlock()
}

// selectAllFromHTTPRoundTrip returns all HTTP round trip events.
func (db *MeasurementDB) selectAllFromHTTPRoundTrip() (out []*HTTPRoundTripEvent) {
	out = append(out, db.httpRoundTripTable...)
	return
}

// InsertIntoHTTPRedirect implements EventDB.InsertIntoHTTPRedirect.
func (db *MeasurementDB) InsertIntoHTTPRedirect(ev *HTTPRedirectEvent) {
	db.mu.Lock()
	db.httpRedirectTable = append(db.httpRedirectTable, ev)
	db.mu.Unlock()
}

// selectAllFromHTTPRedirect returns all HTTP redirections.
func (db *MeasurementDB) selectAllFromHTTPRedirect() (out []*HTTPRedirectEvent) {
	out = append(out, db.httpRedirectTable...)
	return
}

// InsertIntoQUICHandshake implements EventDB.InsertIntoQUICHandshake.
func (db *MeasurementDB) InsertIntoQUICHandshake(ev *QUICHandshakeEvent) {
	db.mu.Lock()
	db.quicHandshakeTable = append(db.quicHandshakeTable, ev)
	db.mu.Unlock()
}

// selectAllFromQUICHandshake returns all QUIC handshake events.
func (db *MeasurementDB) selectAllFromQUICHandshake() (out []*QUICHandshakeEvent) {
	out = append(out, db.quicHandshakeTable...)
	return
}

// AsMeasurement converts the current state of the database into
// a finalized Measurement structure. The original events will remain
// into the database. To start a new measurement cycle, just create
// a new MeasurementDB instance. You are not supposed to modify
// the Measurement returned by this method.
func (db *MeasurementDB) AsMeasurement() *Measurement {
	db.mu.Lock()
	meas := &Measurement{
		Connect:        db.selectAllFromDial(),
		ReadWrite:      db.selectAllFromReadWrite(),
		Close:          db.selectAllFromClose(),
		TLSHandshake:   db.selectAllFromTLSHandshake(),
		QUICHandshake:  db.selectAllFromQUICHandshake(),
		LookupHost:     db.selectAllFromLookupHost(),
		LookupHTTPSSvc: db.selectAllFromLookupHTTPSSvc(),
		DNSRoundTrip:   db.selectAllFromDNSRoundTrip(),
		HTTPRoundTrip:  db.selectAllFromHTTPRoundTrip(),
		HTTPRedirect:   db.selectAllFromHTTPRedirect(),
	}
	db.mu.Unlock()
	return meas
}
