package measurex

// DB is the database holding measurements.
type DB interface {
	// Dial table
	InsertIntoDial(ev *NetworkEvent)
	SelectAllFromDial() []*NetworkEvent

	// ReadWrite table
	InsertIntoReadWrite(ev *NetworkEvent)
	SelectAllFromReadWrite() []*NetworkEvent

	// Close table
	InsertIntoClose(ev *NetworkEvent)
	SelectAllFromClose() []*NetworkEvent

	// TLSHandshake table
	InsertIntoTLSHandshake(ev *TLSHandshakeEvent)
	SelectAllFromTLSHandshake() []*TLSHandshakeEvent

	// LookupHost table
	InsertIntoLookupHost(ev *LookupHostEvent)
	SelectAllFromLookupHost() []*LookupHostEvent

	// LookupHTTPSSvc table
	InsertIntoLookupHTTPSSvc(ev *LookupHTTPSSvcEvent)
	SelectAllFromLookupHTTPSSvc() []*LookupHTTPSSvcEvent

	// DNSRoundTrip table
	InsertIntoDNSRoundTrip(ev *DNSRoundTripEvent)
	SelectAllFromDNSRoundTrip() []*DNSRoundTripEvent

	// HTTPRoundTrip table
	InsertIntoHTTPRoundTrip(ev *HTTPRoundTripEvent)
	SelectAllFromHTTPRoundTrip() []*HTTPRoundTripEvent

	// HTTPRedirect table
	InsertIntoHTTPRedirect(ev *HTTPRedirectEvent)
	SelectAllFromHTTPRedirect() []*HTTPRedirectEvent

	// NextConnID increments and returns the connection ID.
	NextConnID() int64

	// MeasurementID returns the measurement ID.
	MeasurementID() int64

	// NextMeasurement increments and returns the measurement ID.
	NextMeasurement() int64
}
