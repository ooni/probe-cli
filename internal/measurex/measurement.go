package measurex

//
// Measurement
//
// Here we define the fundamental measurement types
// produced by this package.
//

import "time"

// Measurement groups all the events that have the same MeasurementID. This
// data format is not compatible with the OONI data format.
type Measurement struct {
	// MeasurementID is the measurement MeasurementID.
	MeasurementID int64

	// Oddities lists all the oddities inside this measurement. See
	// newMeasurement's docs for more info.
	Oddities []Oddity

	// Connect contains all the connect operations.
	Connect []*NetworkEvent `json:",omitempty"`

	// ReadWrite contains all the read and write operations.
	ReadWrite []*NetworkEvent `json:",omitempty"`

	// Close contains all the close operations.
	Close []*NetworkEvent `json:",omitempty"`

	// TLSHandshake contains all the TLS handshakes.
	TLSHandshake []*TLSHandshakeEvent `json:",omitempty"`

	// QUICHandshake contains all the QUIC handshakes.
	QUICHandshake []*QUICHandshakeEvent `json:",omitempty"`

	// LookupHost contains all the host lookups.
	LookupHost []*LookupHostEvent `json:",omitempty"`

	// LookupHTTPSSvc contains all the HTTPSSvc lookups.
	LookupHTTPSSvc []*LookupHTTPSSvcEvent `json:",omitempty"`

	// DNSRoundTrip contains all the DNS round trips.
	DNSRoundTrip []*DNSRoundTripEvent `json:",omitempty"`

	// HTTPRoundTrip contains all the HTTP round trips.
	HTTPRoundTrip []*HTTPRoundTripEvent `json:",omitempty"`

	// HTTPRedirect contains all the redirections.
	HTTPRedirect []*HTTPRedirectEvent `json:",omitempty"`
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
func NewMeasurement(db *DB, id int64) *Measurement {
	m := &Measurement{
		MeasurementID:  id,
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

// URLMeasurement is the measurement of a whole URL. It contains
// a bunch of measurements detailing each measurement step.
type URLMeasurement struct {
	// URL is the URL we're measuring.
	URL string

	// CannotParseURL is true if the input URL could not be parsed.
	CannotParseURL bool

	// DNS contains all the DNS related measurements.
	DNS []*Measurement

	// TH contains all the measurements from the test helpers.
	TH []*Measurement

	// CannotGenerateEndpoints for URL is true if the code tasked of
	// generating a list of endpoints for the URL fails.
	CannotGenerateEndpoints bool

	// Endpoints contains a measurement for each endpoint
	// that we discovered via DNS or TH.
	Endpoints []*Measurement

	// RedirectURLs contain the URLs to which we should fetch
	// if we choose to follow redirections.
	RedirectURLs []string

	// TotalRuntime is the total time to measure this URL.
	TotalRuntime time.Duration

	// DNSRuntime is the time to run all DNS checks.
	DNSRuntime time.Duration

	// THRuntime is the total time to invoke all test helpers.
	THRuntime time.Duration

	// EpntsRuntime is the total time to check all the endpoints.
	EpntsRuntime time.Duration
}

// fillRedirects takes in input a complete URLMeasurement and fills
// the field named Redirects with all redirections.
func (m *URLMeasurement) fillRedirects() {
	dups := make(map[string]bool)
	for _, epnt := range m.Endpoints {
		for _, redir := range epnt.HTTPRedirect {
			loc := redir.Location.String()
			if _, found := dups[loc]; found {
				continue
			}
			dups[loc] = true
			m.RedirectURLs = append(m.RedirectURLs, loc)
		}
	}
}
