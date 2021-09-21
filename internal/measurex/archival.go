package measurex

import (
	"net/http"
	"strings"
	"unicode/utf8"
)

// ArchivalURLMeasurement is the archival format for URLMeasurement.
type ArchivalURLMeasurement struct {
	URL                     string                 `json:"url"`
	CannotParseURL          bool                   `json:"cannot_parse_url"`
	DNS                     []*ArchivalMeasurement `json:"dns"`
	TH                      []*ArchivalMeasurement `json:"th"`
	CannotGenerateEndpoints bool                   `json:"cannot_generate_endpoints"`
	Endpoints               []*ArchivalMeasurement `json:"endpoints"`
}

// NewArchivalURLMeasurement constructs a new instance
// of the ArchivalURLMeasurement type.
func NewArchivalURLMeasurement(in *URLMeasurement) (out *ArchivalURLMeasurement) {
	return &ArchivalURLMeasurement{
		URL:                     in.URL,
		CannotParseURL:          in.CannotParseURL,
		DNS:                     NewArchivalMeasurementList(in.DNS...),
		TH:                      NewArchivalMeasurementList(in.TH...),
		CannotGenerateEndpoints: in.CannotGenerateEndpoints,
		Endpoints:               NewArchivalMeasurementList(in.Endpoints...),
	}
}

// ArchivalMeasurement is the archival type for Measurement.
type ArchivalMeasurement struct {
	Oddities       []Oddity                    `json:"oddities"`
	Connect        []*ArchivalNetworkEvent     `json:"connect,omitempty"`
	ReadWrite      []*ArchivalNetworkEvent     `json:"read_write,omitempty"`
	TLSHandshake   []*ArchivalTLSQUICHandshake `json:"tls_handshake,omitempty"`
	QUICHandshake  []*ArchivalTLSQUICHandshake `json:"quic_handshake,omitempty"`
	LookupHost     []*ArchivalDNSLookup        `json:"lookup_host,omitempty"`
	LookupHTTPSSvc []*ArchivalDNSLookup        `json:"lookup_httpssvc,omitempty"`
	DNSRoundTrip   []*ArchivalDNSRoundTrip     `json:"dns_round_trip,omitempty"`
	HTTPRoundTrip  []*ArchivalHTTPRoundTrip    `json:"http_round_trip,omitempty"`
}

// NewArchivalMeasurement constructs a new instance
// of the ArchivalMeasurement type.
func NewArchivalMeasurement(in *Measurement) (out *ArchivalMeasurement) {
	return &ArchivalMeasurement{
		Oddities:       in.Oddities,
		Connect:        NewArchivalNetworkEventList(in.Connect...),
		ReadWrite:      NewArchivalNetworkEventList(in.ReadWrite...),
		TLSHandshake:   NewArchivalTLSHandshakeList(in.TLSHandshake...),
		QUICHandshake:  NewArchivalQUICHandshakeList(in.QUICHandshake...),
		LookupHost:     NewArchivalLookupHostList(in.LookupHost...),
		LookupHTTPSSvc: NewArchivalLookupHTTPSSvcList(in.LookupHTTPSSvc...),
		DNSRoundTrip:   NewArchivalDNSRoundTripList(in.DNSRoundTrip...),
		HTTPRoundTrip:  NewArchivalHTTPRoundTripList(in.HTTPRoundTrip...),
	}
}

// NewArchivalMeasurementList takes in input a list of
// Measurement and builds a list of ArchivalMeasurement.
func NewArchivalMeasurementList(in ...*Measurement) (out []*ArchivalMeasurement) {
	for _, m := range in {
		out = append(out, NewArchivalMeasurement(m))
	}
	return
}

// ArchivalNetworkEvent is the data format we use
// to archive all the network events.
type ArchivalNetworkEvent struct {
	// JSON names compatible with df-008-netevents
	RemoteAddr string  `json:"address"`
	ConnID     int64   `json:"conn_id"`
	Error      error   `json:"failure"`
	Count      int     `json:"num_bytes,omitempty"`
	Operation  string  `json:"operation"`
	Network    string  `json:"proto"`
	Finished   float64 `json:"t"`

	// JSON names that are not part of the spec
	Origin  Origin  `json:"origin"`
	Started float64 `json:"started"`
	Oddity  Oddity  `json:"oddity"`
}

// NewArchivalNetworkEvent takes in input a NetworkEvent
// and emits in output an ArchivalNetworkEvent.
func NewArchivalNetworkEvent(in *NetworkEvent) (out *ArchivalNetworkEvent) {
	return &ArchivalNetworkEvent{
		RemoteAddr: in.RemoteAddr,
		ConnID:     in.ConnID,
		Error:      in.Error,
		Count:      in.Count,
		Operation:  in.Operation,
		Network:    in.Network,
		Finished:   in.Finished.Seconds(),
		Origin:     in.Origin,
		Started:    in.Started.Seconds(),
		Oddity:     in.Oddity,
	}
}

// NewArchivalNetworkEventList takes in input a list of
// NetworkEvent and builds a list of ArchivalNetworkEvent.
func NewArchivalNetworkEventList(in ...*NetworkEvent) (out []*ArchivalNetworkEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalNetworkEvent(ev))
	}
	return
}

// ArchivalTLSQUICHandshake is the archival format for TLSHandshakeEvent
// as well as for QUICHandshakeEvent.
type ArchivalTLSQUICHandshake struct {
	// JSON names compatible with df-006-tlshandshake
	CipherSuite     string                `json:"cipher_suite"`
	ConnID          int64                 `json:"conn_id"`
	Error           error                 `json:"failure"`
	NegotiatedProto string                `json:"negotiated_protocol"`
	PeerCerts       []*ArchivalBinaryData `json:"peer_certificates"`
	Finished        float64               `json:"t"`
	TLSVersion      string                `json:"tls_version"`

	// JSON names that are not part of the spec
	Origin     Origin   `json:"origin"`
	Engine     string   `json:"engine"`
	RemoteAddr string   `json:"address"`
	SNI        string   `json:"server_name"` // already used in prod
	ALPN       []string `json:"alpn"`
	SkipVerify bool     `json:"no_tls_verify"` // already used in prod
	Started    float64  `json:"started"`
	Oddity     Oddity   `json:"oddity"`
	Network    string   `json:"network"`
}

// NewArchivalTLSHandshakeList takes in input a list of
// TLSHandshakeEvent and builds a list of ArchivalTLSQUICHandshake.
func NewArchivalTLSHandshakeList(in ...*TLSHandshakeEvent) (out []*ArchivalTLSQUICHandshake) {
	for _, ev := range in {
		out = append(out, NewArchivalTLSHandshake(ev))
	}
	return
}

// NewArchivalTLSHandshake converts a TLSHandshakeEvent to
// its corresponding archival format.
func NewArchivalTLSHandshake(in *TLSHandshakeEvent) (out *ArchivalTLSQUICHandshake) {
	return &ArchivalTLSQUICHandshake{
		CipherSuite:     in.CipherSuite,
		ConnID:          in.ConnID,
		Error:           in.Error,
		NegotiatedProto: in.NegotiatedProto,
		PeerCerts:       NewArchivalTLSCert(in.PeerCerts),
		Finished:        in.Finished.Seconds(),
		TLSVersion:      in.TLSVersion,
		Origin:          in.Origin,
		Engine:          in.Engine,
		RemoteAddr:      in.RemoteAddr,
		SNI:             in.SNI,
		ALPN:            in.ALPN,
		SkipVerify:      in.SkipVerify,
		Started:         in.Started.Seconds(),
		Oddity:          in.Oddity,
		Network:         in.Network,
	}
}

// NewArchivalQUICHandshakeList takes in input a list of
// QUICHandshakeEvent and builds a list of ArchivalTLSQUICHandshake.
func NewArchivalQUICHandshakeList(in ...*QUICHandshakeEvent) (out []*ArchivalTLSQUICHandshake) {
	for _, ev := range in {
		out = append(out, NewArchivalQUICHandshake(ev))
	}
	return
}

// NewArchivalQUICHandshake converts a QUICHandshakeEvent to
// its corresponding archival format.
func NewArchivalQUICHandshake(in *QUICHandshakeEvent) (out *ArchivalTLSQUICHandshake) {
	return &ArchivalTLSQUICHandshake{
		CipherSuite:     in.CipherSuite,
		ConnID:          in.ConnID,
		Error:           in.Error,
		NegotiatedProto: in.NegotiatedProto,
		PeerCerts:       NewArchivalTLSCert(in.PeerCerts),
		Finished:        in.Finished.Seconds(),
		TLSVersion:      in.TLSVersion,
		Origin:          in.Origin,
		RemoteAddr:      in.RemoteAddr,
		SNI:             in.SNI,
		ALPN:            in.ALPN,
		SkipVerify:      in.SkipVerify,
		Started:         in.Started.Seconds(),
		Oddity:          in.Oddity,
		Network:         in.Network,
	}
}

// ArchivalBinaryData is the archival format for binary data.
type ArchivalBinaryData struct {
	Data   []byte
	Format string
}

// NewArchivalTLSCertList builds a new []ArchivalBinaryData
// from a list of raw x509 certificates data.
func NewArchivalTLSCert(in [][]byte) (out []*ArchivalBinaryData) {
	for _, cert := range in {
		out = append(out, &ArchivalBinaryData{
			Data:   cert,
			Format: "base64",
		})
	}
	return
}

// ArchivalDNSLookup is the archival format for DNS.
type ArchivalDNSLookup struct {
	// JSON names compatible with df-002-dnst's spec
	Answers   []*ArchivalDNSAnswer `json:"answers"`
	Network   string               `json:"engine"`
	Error     error                `json:"failure"`
	Domain    string               `json:"hostname"`
	QueryType string               `json:"query_type"`
	Address   string               `json:"resolver_address"`
	Finished  float64              `json:"t"`

	// Names not part of the spec.
	Started float64 `json:"started"`
	Origin  Origin  `json:"origin"`
	Oddity  Oddity  `json:"oddity"`
}

// ArchivalDNSAnswer is an answer inside ArchivalDNS.
type ArchivalDNSAnswer struct {
	// JSON names compatible with df-002-dnst's spec
	Type string `json:"answer_type"`
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ivp6,omitempty"`

	// Names not part of the spec.
	ALPN string `json:"alpn,omitempty"`
}

// NewArchivalLookupHost generates an ArchivalDNS entry for the given
// LookupHost event and for the given query type. (OONI's DNS data
// format splits A and AAAA queries, so we need to run this func twice.)
func NewArchivalLookupHost(in *LookupHostEvent, qtype string) (out *ArchivalDNSLookup) {
	return &ArchivalDNSLookup{
		Answers:   NewArchivalDNSAnswersLookupHost(in.Addrs, qtype),
		Network:   in.Network,
		Error:     in.Error,
		Domain:    in.Domain,
		QueryType: qtype,
		Address:   in.Address,
		Finished:  in.Finished.Seconds(),
		Started:   in.Started.Seconds(),
		Origin:    in.Origin,
		Oddity:    in.Oddity,
	}
}

// NewArchivalDNSAnswersLookupHost builds the ArchivalDNSAnswer
// vector for a LookupHost operation and a given query type.
func NewArchivalDNSAnswersLookupHost(addrs []string, qtype string) (out []*ArchivalDNSAnswer) {
	for _, addr := range addrs {
		switch qtype {
		case "A":
			if !strings.Contains(addr, ":") {
				out = append(out, &ArchivalDNSAnswer{
					Type: qtype,
					IPv4: addr,
				})
			}
		case "AAAA":
			if strings.Contains(addr, ":") {
				out = append(out, &ArchivalDNSAnswer{
					Type: qtype,
					IPv6: addr,
				})
			}
		}
	}
	return
}

// NewArchivalLookupHostList converts a []*LookupHostEvent
// to the corresponding archival format.
func NewArchivalLookupHostList(in ...*LookupHostEvent) (out []*ArchivalDNSLookup) {
	for _, ev := range in {
		out = append(out, NewArchivalLookupHost(ev, "A"))
		out = append(out, NewArchivalLookupHost(ev, "AAAA"))
	}
	return
}

// NewArchivalLookupHTTPSSvc generates an ArchivalDNS entry for the given
// LookupHTTPSSvc event.
func NewArchivalLookupHTTPSSvc(in *LookupHTTPSSvcEvent) (out *ArchivalDNSLookup) {
	return &ArchivalDNSLookup{
		Answers:   NewArchivalDNSAnswersLookupHTTPSSvc(in),
		Network:   in.Network,
		Error:     in.Error,
		Domain:    in.Domain,
		QueryType: "HTTPS",
		Address:   in.Address,
		Finished:  in.Finished.Seconds(),
		Started:   in.Started.Seconds(),
		Origin:    in.Origin,
		Oddity:    in.Oddity,
	}
}

// NewArchivalDNSAnswersLookupHTTPSSvc builds the ArchivalDNSAnswer
// vector for a LookupHTTPSSvc operation.
func NewArchivalDNSAnswersLookupHTTPSSvc(in *LookupHTTPSSvcEvent) (out []*ArchivalDNSAnswer) {
	for _, addr := range in.IPv4 {
		out = append(out, &ArchivalDNSAnswer{
			Type: "A",
			IPv4: addr,
		})
	}
	for _, addr := range in.IPv6 {
		out = append(out, &ArchivalDNSAnswer{
			Type: "AAAA",
			IPv6: addr,
		})
	}
	for _, alpn := range in.ALPN {
		out = append(out, &ArchivalDNSAnswer{
			Type: "ALPN",
			ALPN: alpn,
		})
	}
	return
}

// NewArchivalLookupHTTPSSvcList converts a []*LookupHTTPSSvcEvent
// to the corresponding archival format.
func NewArchivalLookupHTTPSSvcList(in ...*LookupHTTPSSvcEvent) (out []*ArchivalDNSLookup) {
	for _, ev := range in {
		out = append(out, NewArchivalLookupHTTPSSvc(ev))
	}
	return
}

// ArchivalDNSRoundTrip is the archival fromat for DNSRoundTripEvent.
type ArchivalDNSRoundTrip struct {
	Origin   Origin              `json:"origin"`
	Network  string              `json:"engine"`
	Address  string              `json:"resolver_address"`
	Query    *ArchivalBinaryData `json:"raw_query"`
	Started  float64             `json:"started"`
	Finished float64             `json:"t"`
	Error    error               `json:"failure"`
	Reply    *ArchivalBinaryData `json:"raw_reply"`
}

// NewArchivalBinaryData builds a new ArchivalBinaryData
// from an array of bytes. If the array is nil, we return nil.
func NewArchivalBinaryData(data []byte) (out *ArchivalBinaryData) {
	if len(data) > 0 {
		out = &ArchivalBinaryData{
			Data:   data,
			Format: "base64",
		}
	}
	return
}

// NewArchivalDNSRoundTrip converts a DNSRoundTripEvent
// to the corresponding archival format.
func NewArchivalDNSRoundTrip(in *DNSRoundTripEvent) (out *ArchivalDNSRoundTrip) {
	return &ArchivalDNSRoundTrip{
		Origin:   in.Origin,
		Network:  in.Network,
		Address:  in.Address,
		Query:    NewArchivalBinaryData(in.Query),
		Started:  in.Started.Seconds(),
		Finished: in.Finished.Seconds(),
		Error:    in.Error,
		Reply:    NewArchivalBinaryData(in.Reply),
	}
}

// NewArchivalDNSRoundTripList converts a []*DNSRoundTripEvent
// to the corresponding archival format.
func NewArchivalDNSRoundTripList(in ...*DNSRoundTripEvent) (out []*ArchivalDNSRoundTrip) {
	for _, ev := range in {
		out = append(out, NewArchivalDNSRoundTrip(ev))
	}
	return
}

// ArchivalHTTPRoundTrip is the archival format for HTTPRoundTripEvent.
type ArchivalHTTPRoundTrip struct {
	// JSON names following the df-001-httpt data format.
	Error    error                 `json:"failure"`
	Request  *ArchivalHTTPRequest  `json:"request"`
	Response *ArchivalHTTPResponse `json:"response"`
	Finished float64               `json:"t"`
	ConnID   int64                 `json:"conn_id"`
	Started  float64               `json:"started"`

	// Names not in the specification
	Origin Origin `json:"origin"`
	Oddity Oddity `json:"oddity"`
}

// ArchivalHTTPRequest is the archival representation of a request.
type ArchivalHTTPRequest struct {
	Method      string     `json:"method"`
	URL         string     `json:"url"`
	HeadersList [][]string `json:"headers_list"`
}

// ArchivalHTTPResponse is the archival representation of a response.
type ArchivalHTTPResponse struct {
	Code            int64       `json:"code"`
	HeadersList     [][]string  `json:"headers_list"`
	Body            interface{} `json:"body"`
	BodyIsTruncated bool        `json:"body_is_truncated"`
}

// NewArchivalHTTPRoundTrip converts an HTTPRoundTripEvent
// to the corresponding archival format.
func NewArchivalHTTPRoundTrip(in *HTTPRoundTripEvent) (out *ArchivalHTTPRoundTrip) {
	return &ArchivalHTTPRoundTrip{
		Error: in.Error,
		Request: &ArchivalHTTPRequest{
			Method:      in.RequestMethod,
			URL:         in.RequestURL.String(),
			HeadersList: NewArchivalHeadersList(in.RequestHeader),
		},
		Response: &ArchivalHTTPResponse{
			Code:            int64(in.ResponseStatus),
			HeadersList:     NewArchivalHeadersList(in.ResponseHeader),
			Body:            NewArchivalHTTPBody(in.ResponseBodySnapshot),
			BodyIsTruncated: int64(len(in.ResponseBodySnapshot)) >= in.MaxBodySnapshotSize,
		},
		Finished: in.Finished.Seconds(),
		ConnID:   in.ConnID,
		Started:  in.Started.Seconds(),
		Origin:   in.Origin,
		Oddity:   in.Oddity,
	}
}

// NewArchivalHTTPBody builds a new HTTP body for archival from the body.
func NewArchivalHTTPBody(body []byte) interface{} {
	if utf8.Valid(body) {
		return string(body)
	}
	return &ArchivalBinaryData{
		Data:   body,
		Format: "base64",
	}
}

// NewArchivalHeadersList builds a new HeadersList from http.Header.
func NewArchivalHeadersList(in http.Header) (out [][]string) {
	for k, vv := range in {
		for _, v := range vv {
			out = append(out, []string{k, v})
		}
	}
	return
}

// NewArchivalHTTPRoundTripList converts a []*HTTPRoundTripEvent
// to the corresponding archival format.
func NewArchivalHTTPRoundTripList(in ...*HTTPRoundTripEvent) (out []*ArchivalHTTPRoundTrip) {
	for _, ev := range in {
		out = append(out, NewArchivalHTTPRoundTrip(ev))
	}
	return
}
