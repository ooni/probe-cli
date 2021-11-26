package measurex

//
// DB
//
// This file defines two types:
//
// - WritableDB is the interface allowing networking code
// (e.g., Dialer to save measurement events);
//
// - MeasurementDB implements WritableDB and allows high-level
// code to generate a Measurement from all the events.
//

import "sync"

// WritableDB is an events "database" in which networking code
// (e.g., Dialer) can save measurement events (e.g., the result
// of a connect, a TLS handshake, a read).
type WritableDB interface {
	// InsertIntoDial saves a Dial event.
	InsertIntoDial(ev *NetworkEvent)

	// InsertIntoReadWrite saves an I/O event.
	InsertIntoReadWrite(ev *NetworkEvent)

	// InsertIntoClose saves a close event.
	InsertIntoClose(ev *NetworkEvent)

	// InsertIntoTLSHandshake saves a TLS handshake event.
	InsertIntoTLSHandshake(ev *QUICTLSHandshakeEvent)

	// InsertIntoLookupHost saves a lookup host event.
	InsertIntoLookupHost(ev *DNSLookupEvent)

	// InsertIntoLookupHTTPSvc saves an HTTPSvc lookup event.
	InsertIntoLookupHTTPSSvc(ev *DNSLookupEvent)

	// InsertIntoDNSRoundTrip saves a DNS round trip event.
	InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent)

	// InsertIntoHTTPRoundTrip saves an HTTP round trip event.
	InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent)

	// InsertIntoHTTPRedirect saves an HTTP redirect event.
	InsertIntoHTTPRedirect(ev *HTTPRedirectEvent)

	// InsertIntoQUICHandshake saves a QUIC handshake event.
	InsertIntoQUICHandshake(ev *QUICTLSHandshakeEvent)
}

// MeasurementDB is a WritableDB that also allows high-level code
// to generate a Measurement from all the saved events.
type MeasurementDB struct {
	// database "tables"
	dialTable          []*NetworkEvent
	readWriteTable     []*NetworkEvent
	closeTable         []*NetworkEvent
	tlsHandshakeTable  []*QUICTLSHandshakeEvent
	lookupHostTable    []*DNSLookupEvent
	lookupHTTPSvcTable []*DNSLookupEvent
	dnsRoundTripTable  []*DNSRoundTripEvent
	httpRoundTripTable []*HTTPRoundTripEvent
	httpRedirectTable  []*HTTPRedirectEvent
	quicHandshakeTable []*QUICTLSHandshakeEvent

	// mu protects all the fields
	mu sync.Mutex
}

var _ WritableDB = &MeasurementDB{}

// DeleteAll deletes all the content of the DB.
func (db *MeasurementDB) DeleteAll() {
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
func (db *MeasurementDB) InsertIntoDial(ev *NetworkEvent) {
	db.mu.Lock()
	db.dialTable = append(db.dialTable, ev)
	db.mu.Unlock()
}

// selectAllFromDialUnlocked returns all dial events.
func (db *MeasurementDB) selectAllFromDialUnlocked() (out []*NetworkEvent) {
	out = append(out, db.dialTable...)
	return
}

// InsertIntoReadWrite implements EventDB.InsertIntoReadWrite.
func (db *MeasurementDB) InsertIntoReadWrite(ev *NetworkEvent) {
	db.mu.Lock()
	db.readWriteTable = append(db.readWriteTable, ev)
	db.mu.Unlock()
}

// selectAllFromReadWriteUnlocked returns all I/O events.
func (db *MeasurementDB) selectAllFromReadWriteUnlocked() (out []*NetworkEvent) {
	out = append(out, db.readWriteTable...)
	return
}

// InsertIntoClose implements EventDB.InsertIntoClose.
func (db *MeasurementDB) InsertIntoClose(ev *NetworkEvent) {
	db.mu.Lock()
	db.closeTable = append(db.closeTable, ev)
	db.mu.Unlock()
}

// selectAllFromCloseUnlocked returns all close events.
func (db *MeasurementDB) selectAllFromCloseUnlocked() (out []*NetworkEvent) {
	out = append(out, db.closeTable...)
	return
}

// InsertIntoTLSHandshake implements EventDB.InsertIntoTLSHandshake.
func (db *MeasurementDB) InsertIntoTLSHandshake(ev *QUICTLSHandshakeEvent) {
	db.mu.Lock()
	db.tlsHandshakeTable = append(db.tlsHandshakeTable, ev)
	db.mu.Unlock()
}

// selectAllFromTLSHandshakeUnlocked returns all TLS handshake events.
func (db *MeasurementDB) selectAllFromTLSHandshakeUnlocked() (out []*QUICTLSHandshakeEvent) {
	out = append(out, db.tlsHandshakeTable...)
	return
}

// InsertIntoLookupHost implements EventDB.InsertIntoLookupHost.
func (db *MeasurementDB) InsertIntoLookupHost(ev *DNSLookupEvent) {
	db.mu.Lock()
	db.lookupHostTable = append(db.lookupHostTable, ev)
	db.mu.Unlock()
}

// selectAllFromLookupHostUnlocked returns all the lookup host events.
func (db *MeasurementDB) selectAllFromLookupHostUnlocked() (out []*DNSLookupEvent) {
	out = append(out, db.lookupHostTable...)
	return
}

// InsertIntoHTTPSSvc implements EventDB.InsertIntoHTTPSSvc
func (db *MeasurementDB) InsertIntoLookupHTTPSSvc(ev *DNSLookupEvent) {
	db.mu.Lock()
	db.lookupHTTPSvcTable = append(db.lookupHTTPSvcTable, ev)
	db.mu.Unlock()
}

// selectAllFromLookupHTTPSSvcUnlocked returns all HTTPSSvc lookup events.
func (db *MeasurementDB) selectAllFromLookupHTTPSSvcUnlocked() (out []*DNSLookupEvent) {
	out = append(out, db.lookupHTTPSvcTable...)
	return
}

// InsertIntoDNSRoundTrip implements EventDB.InsertIntoDNSRoundTrip.
func (db *MeasurementDB) InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent) {
	db.mu.Lock()
	db.dnsRoundTripTable = append(db.dnsRoundTripTable, ev)
	db.mu.Unlock()
}

// selectAllFromDNSRoundTripUnlocked returns all DNS round trip events.
func (db *MeasurementDB) selectAllFromDNSRoundTripUnlocked() (out []*DNSRoundTripEvent) {
	out = append(out, db.dnsRoundTripTable...)
	return
}

// InsertIntoHTTPRoundTrip implements EventDB.InsertIntoHTTPRoundTrip.
func (db *MeasurementDB) InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent) {
	db.mu.Lock()
	db.httpRoundTripTable = append(db.httpRoundTripTable, ev)
	db.mu.Unlock()
}

// selectAllFromHTTPRoundTripUnlocked returns all HTTP round trip events.
func (db *MeasurementDB) selectAllFromHTTPRoundTripUnlocked() (out []*HTTPRoundTripEvent) {
	out = append(out, db.httpRoundTripTable...)
	return
}

// InsertIntoHTTPRedirect implements EventDB.InsertIntoHTTPRedirect.
func (db *MeasurementDB) InsertIntoHTTPRedirect(ev *HTTPRedirectEvent) {
	db.mu.Lock()
	db.httpRedirectTable = append(db.httpRedirectTable, ev)
	db.mu.Unlock()
}

// selectAllFromHTTPRedirectUnlocked returns all HTTP redirections.
func (db *MeasurementDB) selectAllFromHTTPRedirectUnlocked() (out []*HTTPRedirectEvent) {
	out = append(out, db.httpRedirectTable...)
	return
}

// InsertIntoQUICHandshake implements EventDB.InsertIntoQUICHandshake.
func (db *MeasurementDB) InsertIntoQUICHandshake(ev *QUICTLSHandshakeEvent) {
	db.mu.Lock()
	db.quicHandshakeTable = append(db.quicHandshakeTable, ev)
	db.mu.Unlock()
}

// selectAllFromQUICHandshakeUnlocked returns all QUIC handshake events.
func (db *MeasurementDB) selectAllFromQUICHandshakeUnlocked() (out []*QUICTLSHandshakeEvent) {
	out = append(out, db.quicHandshakeTable...)
	return
}

// AsMeasurement converts the current state of the database into
// a finalized Measurement structure. The original events will remain
// into the database. To start a new measurement cycle, just create
// a new MeasurementDB instance and use that.
func (db *MeasurementDB) AsMeasurement() *Measurement {
	db.mu.Lock()
	meas := &Measurement{
		Connect:        db.selectAllFromDialUnlocked(),
		ReadWrite:      db.selectAllFromReadWriteUnlocked(),
		Close:          db.selectAllFromCloseUnlocked(),
		TLSHandshake:   db.selectAllFromTLSHandshakeUnlocked(),
		QUICHandshake:  db.selectAllFromQUICHandshakeUnlocked(),
		LookupHost:     db.selectAllFromLookupHostUnlocked(),
		LookupHTTPSSvc: db.selectAllFromLookupHTTPSSvcUnlocked(),
		DNSRoundTrip:   db.selectAllFromDNSRoundTripUnlocked(),
		HTTPRoundTrip:  db.selectAllFromHTTPRoundTripUnlocked(),
		HTTPRedirect:   db.selectAllFromHTTPRedirectUnlocked(),
	}
	db.mu.Unlock()
	return meas
}
