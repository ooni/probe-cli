package measurex

import "time"

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
