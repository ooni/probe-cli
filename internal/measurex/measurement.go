package measurex

// Measurement groups all the events that have the same MeasurementID. This
// data format is not compatible with the OONI data format.
type Measurement struct {
	// ID is the measurement ID.
	ID int64

	// URL is the OPTIONAL URL this measurement refers to.
	URL string

	// Endpoint is the OPTIONAL endpoint this measurement refers to.
	Endpoint string

	// Oddities lists all the oddities inside this measurement. See
	// newMeasurement's docs for more info.
	Oddities []Oddity

	// Connect contains all the connect operations.
	Connect []*NetworkEvent

	// ReadWrite contains all the read and write operations.
	ReadWrite []*NetworkEvent

	// Close contains all the close operations.
	Close []*NetworkEvent

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*TLSHandshakeEvent

	// QUICHandshake contains all the QUIC handshakes.
	QUICHandshake []*QUICHandshakeEvent

	// LookupHost contains all the host lookups.
	LookupHost []*LookupHostEvent

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*LookupHTTPSSvcEvent

	// DNSRoundTrip contains all the DNS round trips.
	DNSRoundTrip []*DNSRoundTripEvent

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*HTTPRoundTripEvent

	// HTTPRedirect contains all the redirections.
	HTTPRedirect []*HTTPRedirectEvent
}

// NewMeasurement creates a new Measurement by gathering all the
// events inside the database with a given MeasurementID.
//
// As part of the process, this function computes the Oddities field by
// gathering the oddities of the following operations:
//
// - connect;
//
// - tlsHandshake;
//
// - quicHandshake;
//
// - lookupHost;
//
// - httpRoundTrip.
//
// Arguments:
//
// - begin is the time when we started measuring;
//
// - id is the MeasurementID.
//
// Returns a Measurement possibly containing empty lists of events.
func NewMeasurement(db *Saver, id int64) *Measurement {
	m := &Measurement{
		ID:             id,
		Connect:        db.SelectAllFromDialWithMeasurementID(id),
		ReadWrite:      db.SelectAllFromReadWriteWithMeasurementID(id),
		Close:          db.SelectAllFromCloseWithMeasurementID(id),
		TLSHandshake:   db.SelectAllFromTLSHandshakeWithMeasurementID(id),
		QUICHandshake:  db.SelectAllFromQUICHandshakeWithMeasurementID(id),
		LookupHost:     db.SelectAllFromLookupHostWithMeasurementID(id),
		LookupHTTPSSvc: db.SelectAllFromLookupHTTPSSvcWithMeasurementID(id),
		DNSRoundTrip:   db.SelectAllFromDNSRoundTripWithMeasurementID(id),
		HTTPRoundTrip:  db.SelectAllFromHTTPRoundTripWithMeasurementID(id),
		HTTPRedirect:   db.SelectAllFromHTTPRedirectWithMeasurementID(id),
	}
	m.computeOddities()
	return m
}

// computeOddities computes all the oddities inside m. See
// newMeasurement's docs for more information.
func (m *Measurement) computeOddities() {
	unique := make(map[Oddity]bool)
	for _, ev := range m.Connect {
		unique[ev.Oddity] = true
	}
	for _, ev := range m.TLSHandshake {
		unique[ev.Oddity] = true
	}
	for _, ev := range m.QUICHandshake {
		unique[ev.Oddity] = true
	}
	for _, ev := range m.LookupHost {
		unique[ev.Oddity] = true
	}
	for _, ev := range m.HTTPRoundTrip {
		unique[ev.Oddity] = true
	}
	for key := range unique {
		if key != "" {
			m.Oddities = append(m.Oddities, key)
		}
	}
}
