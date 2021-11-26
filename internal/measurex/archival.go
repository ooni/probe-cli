package measurex

import (
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

//
// Archival
//
// This file defines helpers to serialize to the OONI data format.
//

//
// BinaryData
//

// ArchivalBinaryData is the archival format for binary data.
type ArchivalBinaryData struct {
	Data   []byte `json:"data"`
	Format string `json:"format"`
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

//
// NetworkEvent
//

// ArchivalNetworkEvent is the OONI data format representation
// of a network event according to df-008-netevents.
type ArchivalNetworkEvent struct {
	// JSON names compatible with df-008-netevents
	RemoteAddr string  `json:"address"`
	Failure    *string `json:"failure"`
	Count      int     `json:"num_bytes,omitempty"`
	Operation  string  `json:"operation"`
	Network    string  `json:"proto"`
	Finished   float64 `json:"t"`
	Started    float64 `json:"started"`

	// Names that are not part of the spec.
	Oddity Oddity `json:"oddity"`
}

// NewArchivalNetworkEvent converts a network event to its archival format.
func NewArchivalNetworkEvent(in *NetworkEvent) *ArchivalNetworkEvent {
	return &ArchivalNetworkEvent{
		RemoteAddr: in.RemoteAddr,
		Failure:    in.Failure,
		Count:      in.Count,
		Operation:  in.Operation,
		Network:    in.Network,
		Finished:   in.Finished,
		Started:    in.Started,
		Oddity:     in.Oddity,
	}
}

// NewArchivalNetworkEventList converts a list of NetworkEvent
// to a list of ArchivalNetworkEvent.
func NewArchivalNetworkEventList(in []*NetworkEvent) (out []*ArchivalNetworkEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalNetworkEvent(ev))
	}
	return
}

//
// DNSRoundTripEvent
//

// ArchivalDNSRoundTripEvent is the OONI data format representation
// of a DNS round trip, which is currently not specified.
//
// We are trying to use names compatible with the names currently
// used by other specifications we currently use.
type ArchivalDNSRoundTripEvent struct {
	Network  string              `json:"engine"`
	Address  string              `json:"resolver_address"`
	Query    *ArchivalBinaryData `json:"raw_query"`
	Started  float64             `json:"started"`
	Finished float64             `json:"t"`
	Failure  *string             `json:"failure"`
	Reply    *ArchivalBinaryData `json:"raw_reply"`
}

// NewArchivalDNSRoundTripEvent converts a DNSRoundTripEvent into is archival format.
func NewArchivalDNSRoundTripEvent(in *DNSRoundTripEvent) *ArchivalDNSRoundTripEvent {
	return &ArchivalDNSRoundTripEvent{
		Network:  in.Network,
		Address:  in.Address,
		Query:    NewArchivalBinaryData(in.Query),
		Started:  in.Started,
		Finished: in.Finished,
		Failure:  in.Failure,
		Reply:    NewArchivalBinaryData(in.Reply),
	}
}

// NewArchivalDNSRoundTripEventList converts a DNSRoundTripEvent
// list to the corresponding archival format.
func NewArchivalDNSRoundTripEventList(in []*DNSRoundTripEvent) (out []*ArchivalDNSRoundTripEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalDNSRoundTripEvent(ev))
	}
	return
}

//
// HTTPRoundTrip
//

// ArchivalHTTPRequest is the archival format of an HTTP
// request according to df-001-http.md.
type ArchivalHTTPRequest struct {
	Method  string          `json:"method"`
	URL     string          `json:"url"`
	Headers ArchivalHeaders `json:"headers"`
}

// ArchivalHTTPResponse is the archival format of an HTTP
// response according to df-001-http.md.
type ArchivalHTTPResponse struct {
	// Names consistent with df-001-http.md
	Code            int64               `json:"code"`
	Headers         ArchivalHeaders     `json:"headers"`
	Body            *ArchivalBinaryData `json:"body"`
	BodyIsTruncated bool                `json:"body_is_truncated"`

	// Fields not part of the spec
	BodyLength int64 `json:"x_body_length"`
	BodyIsUTF8 bool  `json:"x_body_is_utf8"`
}

// ArchivalHTTPRoundTripEvent is the archival format of an
// HTTP response according to df-001-http.md.
type ArchivalHTTPRoundTripEvent struct {
	// JSON names following the df-001-httpt data format.
	Failure  *string       `json:"failure"`
	Request  *HTTPRequest  `json:"request"`
	Response *HTTPResponse `json:"response"`
	Finished float64       `json:"t"`
	Started  float64       `json:"started"`

	// Names not in the specification
	Oddity Oddity `json:"oddity"`
}

// ArchivalHeaders is a list of HTTP headers.
type ArchivalHeaders map[string]string

// Get searches for the first header with the named key
// and returns it. If not found, returns an empty string.
func (headers ArchivalHeaders) Get(key string) string {
	return headers[strings.ToLower(key)]
}

// NewArchivalHeaders builds a new HeadersList from http.Header.
func NewArchivalHeaders(in http.Header) (out ArchivalHeaders) {
	out = make(ArchivalHeaders)
	for k, vv := range in {
		for _, v := range vv {
			// It breaks my hearth a little bit to ignore
			// subsequent headers, but this does not happen
			// very frequently, and I know the pipeline
			// parses the map headers format only.
			out[strings.ToLower(k)] = v
			break
		}
	}
	return
}

// NewArchivalHTTPRoundTripEvent converts an HTTPRoundTrip to its archival format.
func NewArchivalHTTPRoundTripEvent(in *HTTPRoundTripEvent) *ArchivalHTTPRoundTripEvent {
	return &ArchivalHTTPRoundTripEvent{
		Failure: in.Failure,
		Request: &HTTPRequest{
			Method:  in.Method,
			URL:     in.URL,
			Headers: NewArchivalHeaders(in.RequestHeaders),
		},
		Response: &HTTPResponse{
			Code:            in.StatusCode,
			Headers:         NewArchivalHeaders(in.ResponseHeaders),
			Body:            NewArchivalBinaryData(in.ResponseBody),
			BodyLength:      in.ResponseBodyLength,
			BodyIsTruncated: in.ResponseBodyIsTruncated,
			BodyIsUTF8:      in.ResponseBodyIsUTF8,
		},
		Finished: in.Finished,
		Started:  in.Started,
		Oddity:   in.Oddity,
	}
}

// NewArchivalHTTPRoundTripEventList converts a list of
// HTTPRoundTripEvent to a list of ArchivalRoundTripEvent.
func NewArchivalHTTPRoundTripEventList(in []*HTTPRoundTripEvent) (out []*ArchivalHTTPRoundTripEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalHTTPRoundTripEvent(ev))
	}
	return
}

//
// QUICTLSHandshakeEvent
//

// ArchivalQUICTLSHandshakeEvent is the archival data format for a
// QUIC or TLS handshake event according to df-006-tlshandshake.
type ArchivalQUICTLSHandshakeEvent struct {
	// JSON names compatible with df-006-tlshandshake
	CipherSuite     string                `json:"cipher_suite"`
	Failure         *string               `json:"failure"`
	NegotiatedProto string                `json:"negotiated_proto"`
	TLSVersion      string                `json:"tls_version"`
	PeerCerts       []*ArchivalBinaryData `json:"peer_certificates"`
	Finished        float64               `json:"t"`

	// JSON names that are consistent with the
	// spirit of the spec but are not in it
	RemoteAddr string   `json:"address"`
	SNI        string   `json:"server_name"` // used in prod
	ALPN       []string `json:"alpn"`
	SkipVerify bool     `json:"no_tls_verify"` // used in prod
	Oddity     Oddity   `json:"oddity"`
	Network    string   `json:"proto"`
	Started    float64  `json:"started"`
}

// NewArchivalTLSCertList builds a new []ArchivalBinaryData
// from a list of raw x509 certificates data.
func NewArchivalTLSCerts(in [][]byte) (out []*ArchivalBinaryData) {
	for _, cert := range in {
		out = append(out, &ArchivalBinaryData{
			Data:   cert,
			Format: "base64",
		})
	}
	return
}

// NewArchivalQUICTLSHandshakeEvent converts a QUICTLSHandshakeEvent
// to its archival data format.
func NewArchivalQUICTLSHandshakeEvent(in *QUICTLSHandshakeEvent) *ArchivalQUICTLSHandshakeEvent {
	return &ArchivalQUICTLSHandshakeEvent{
		CipherSuite:     in.CipherSuite,
		Failure:         in.Failure,
		NegotiatedProto: in.NegotiatedProto,
		TLSVersion:      in.TLSVersion,
		PeerCerts:       NewArchivalTLSCerts(in.PeerCerts),
		Finished:        in.Finished,
		RemoteAddr:      in.RemoteAddr,
		SNI:             in.SNI,
		ALPN:            in.ALPN,
		SkipVerify:      in.SkipVerify,
		Oddity:          in.Oddity,
		Network:         in.Network,
		Started:         in.Started,
	}
}

// NewArchivalQUICTLSHandshakeEventList converts a list of
// QUICTLSHandshakeEvent to a list of ArchivalQUICTLSHandshakeEvent.
func NewArchivalQUICTLSHandshakeEventList(in []*QUICTLSHandshakeEvent) (out []*ArchivalQUICTLSHandshakeEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalQUICTLSHandshakeEvent(ev))
	}
	return
}

//
// DNSLookup
//

// ArchivalDNSLookupAnswer is the archival format of a
// DNS lookup answer according to df-002-dnst.
type ArchivalDNSLookupAnswer struct {
	// JSON names compatible with df-002-dnst's spec
	Type string `json:"answer_type"`
	IPv4 string `json:"ipv4,omitempty"`
	IPv6 string `json:"ivp6,omitempty"`

	// Names not part of the spec.
	ALPN string `json:"alpn,omitempty"`
}

// ArchivalDNSLookupEvent is the archival data format
// of a DNS lookup according to df-002-dnst.
type ArchivalDNSLookupEvent struct {
	// fields inside df-002-dnst
	Answers   []ArchivalDNSLookupAnswer `json:"answers"`
	Network   string                    `json:"engine"`
	Failure   *string                   `json:"failure"`
	Domain    string                    `json:"hostname"`
	QueryType string                    `json:"query_type"`
	Address   string                    `json:"resolver_address"`
	Finished  float64                   `json:"t"`

	// Names not part of the spec.
	Started float64 `json:"started"`
	Oddity  Oddity  `json:"oddity"`
}

// NewArchivalDNSLookupAnswers creates a list of ArchivalDNSLookupAnswer.
func NewArchivalDNSLookupAnswers(in *DNSLookupEvent) (out []ArchivalDNSLookupAnswer) {
	for _, ip := range in.A {
		out = append(out, ArchivalDNSLookupAnswer{
			Type: "A",
			IPv4: ip,
		})
	}
	for _, ip := range in.AAAA {
		out = append(out, ArchivalDNSLookupAnswer{
			Type: "AAAA",
			IPv6: ip,
		})
	}
	for _, alpn := range in.ALPN {
		out = append(out, ArchivalDNSLookupAnswer{
			Type: "ALPN",
			ALPN: alpn,
		})
	}
	return
}

// NewArchivalDNSLookupEvent converts a DNSLookupEvent
// to its archival representation.
func NewArchivalDNSLookupEvent(in *DNSLookupEvent) *ArchivalDNSLookupEvent {
	return &ArchivalDNSLookupEvent{
		Answers:   NewArchivalDNSLookupAnswers(in),
		Network:   in.Network,
		Failure:   in.Failure,
		Domain:    in.Domain,
		QueryType: in.QueryType,
		Address:   in.Address,
		Finished:  in.Finished,
		Started:   in.Started,
		Oddity:    in.Oddity,
	}
}

// NewArchivalDNSLookupEventList converts a list of DNSLookupEvent
// to a list of ArchivalDNSLookupEvent.
func NewArchivalDNSLookupEventList(in []*DNSLookupEvent) (out []*ArchivalDNSLookupEvent) {
	for _, ev := range in {
		out = append(out, NewArchivalDNSLookupEvent(ev))
	}
	return
}

//
// TCPConnect
//

// ArchivalTCPConnect is the archival form of TCP connect
// events in compliance with df-005-tcpconnect.
type ArchivalTCPConnect struct {
	// Names part of the spec.
	IP       string                    `json:"ip"`
	Port     int64                     `json:"port"`
	Finished float64                   `json:"t"`
	Status   *ArchivalTCPConnectStatus `json:"status"`

	// Names not part of the spec.
	Started float64 `json:"started"`
	Oddity  Oddity  `json:"oddity"`
}

// ArchivalTCPConnectStatus contains the status of a TCP connect.
type ArchivalTCPConnectStatus struct {
	Blocked bool    `json:"blocked"`
	Failure *string `json:"failure"`
	Success bool    `json:"success"`
}

// NewArchivalTCPConnect converts a NetworkEvent to an ArchivalTCPConnect.
func NewArchivalTCPConnect(in *NetworkEvent) *ArchivalTCPConnect {
	// We ignore errors because values come from Go code that
	// emits correct serialization of TCP/UDP addresses.
	addr, port, _ := net.SplitHostPort(in.RemoteAddr)
	portnum, _ := strconv.Atoi(port)
	return &ArchivalTCPConnect{
		IP:       addr,
		Port:     int64(portnum),
		Finished: in.Finished,
		Status: &ArchivalTCPConnectStatus{
			Blocked: in.Failure != nil,
			Failure: in.Failure,
			Success: in.Failure == nil,
		},
		Started: in.Started,
		Oddity:  in.Oddity,
	}
}

// NewArchivalTCPConnectList converts a list of NetworkEvent
// to a list of ArchivalTCPConnect. In doing that, the code
// only considers "connect" events using the TCP protocol.
func NewArchivalTCPConnectList(in []*NetworkEvent) (out []*ArchivalTCPConnect) {
	for _, ev := range in {
		if ev.Operation != "connect" {
			continue
		}
		switch ev.Network {
		case "tcp", "tcp4", "tcp6":
			out = append(out, NewArchivalTCPConnect(ev))
		default:
			// nothing
		}
	}
	return
}

//
// URLMeasurement
//

// ArchivalURLMeasurement is the archival representation of URLMeasurement
type ArchivalURLMeasurement struct {
	URL          string                             `json:"url"`
	DNS          []*ArchivalDNSMeasurement          `json:"dns"`
	Endpoints    []*ArchivalHTTPEndpointMeasurement `json:"endpoints"`
	TH           *ArchivalTHMeasurement             `json:"th"`
	TotalRuntime time.Duration                      `json:"x_total_runtime"`
	DNSRuntime   time.Duration                      `json:"x_dns_runtime"`
	THRuntime    time.Duration                      `json:"x_th_runtime"`
	EpntsRuntime time.Duration                      `json:"x_epnts_runtime"`
}

// NewArchivalURLMeasurement creates the archival representation
// of an URLMeasurement data structure.
func NewArchivalURLMeasurement(in *URLMeasurement) *ArchivalURLMeasurement {
	return &ArchivalURLMeasurement{
		URL:          in.URL,
		DNS:          NewArchivalDNSMeasurementList(in.DNS),
		Endpoints:    NewArchivalHTTPEndpointMeasurementList(in.Endpoints),
		TH:           NewArchivalTHMeasurement(in.TH),
		TotalRuntime: in.TotalRuntime,
		DNSRuntime:   in.DNSRuntime,
		THRuntime:    in.THRuntime,
		EpntsRuntime: in.EpntsRuntime,
	}
}

//
// EndpointMeasurement
//

// ArchivalEndpointMeasurement is the archival representation of EndpointMeasurement.
type ArchivalEndpointMeasurement struct {
	// Network is the network of this endpoint.
	Network EndpointNetwork `json:"network"`

	// Address is the address of this endpoint.
	Address string `json:"address"`

	// An EndpointMeasurement is a Measurement.
	*ArchivalMeasurement
}

// NewArchivalEndpointMeasurement converts an EndpointMeasurement
// to the corresponding archival data format.
func NewArchivalEndpointMeasurement(in *EndpointMeasurement) *ArchivalEndpointMeasurement {
	return &ArchivalEndpointMeasurement{
		Network:             in.Network,
		Address:             in.Address,
		ArchivalMeasurement: NewArchivalMeasurement(in.Measurement),
	}
}

//
// THMeasurement
//

// ArchivalTHMeasurement is the archival representation of THMeasurement.
type ArchivalTHMeasurement struct {
	DNS       []*ArchivalDNSMeasurement          `json:"dns"`
	Endpoints []*ArchivalHTTPEndpointMeasurement `json:"endpoints"`
}

// NewArchivalTHMeasurement creates the archival representation of THMeasurement.
func NewArchivalTHMeasurement(in *THMeasurement) (out *ArchivalTHMeasurement) {
	if in != nil {
		out = &ArchivalTHMeasurement{
			DNS:       NewArchivalDNSMeasurementList(in.DNS),
			Endpoints: NewArchivalHTTPEndpointMeasurementList(in.Endpoints),
		}
	}
	return
}

//
// DNSMeasurement
//

// ArchivalDNSMeasurement is the archival representation of DNSMeasurement.
type ArchivalDNSMeasurement struct {
	Domain string `json:"domain"`
	*ArchivalMeasurement
}

// NewArchivalDNSMeasurement converts a DNSMeasurement to an ArchivalDNSMeasurement.
func NewArchivalDNSMeasurement(in *DNSMeasurement) *ArchivalDNSMeasurement {
	return &ArchivalDNSMeasurement{
		Domain:              in.Domain,
		ArchivalMeasurement: NewArchivalMeasurement(in.Measurement),
	}
}

// NewArchivalDNSMeasurementList converts a list of DNSMeasurement
// to a list of ArchivalDNSMeasurement.
func NewArchivalDNSMeasurementList(in []*DNSMeasurement) (out []*ArchivalDNSMeasurement) {
	for _, m := range in {
		out = append(out, NewArchivalDNSMeasurement(m))
	}
	return
}

//
// HTTPEndpointMeasurement
//

// ArchivalHTTPEndpointMeasurement is the archival representation
// of an HTTPEndpointMeasurement.
type ArchivalHTTPEndpointMeasurement struct {
	URL     string          `json:"url"`
	Network EndpointNetwork `json:"network"`
	Address string          `json:"address"`
	*ArchivalMeasurement
}

// NewArchivalHTTPEndpointMeasurement converts an HTTPEndpointMeasurement
// to an ArchivalHTTPEndpointMeasurement.
func NewArchivalHTTPEndpointMeasurement(in *HTTPEndpointMeasurement) *ArchivalHTTPEndpointMeasurement {
	return &ArchivalHTTPEndpointMeasurement{
		URL:                 in.URL,
		Network:             in.Network,
		Address:             in.Address,
		ArchivalMeasurement: NewArchivalMeasurement(in.Measurement),
	}
}

// NewArchivalHTTPEndpointMeasurementList converts a list of HTTPEndpointMeasurement
// to a list of ArchivalHTTPEndpointMeasurement.
func NewArchivalHTTPEndpointMeasurementList(in []*HTTPEndpointMeasurement) (out []*ArchivalHTTPEndpointMeasurement) {
	for _, m := range in {
		out = append(out, NewArchivalHTTPEndpointMeasurement(m))
	}
	return
}

//
// Measurement
//

// ArchivalMeasurement is the archival representation of a Measurement.
type ArchivalMeasurement struct {
	NetworkEvents  []*ArchivalNetworkEvent          `json:"network_events,omitempty"`
	DNSEvents      []*ArchivalDNSRoundTripEvent     `json:"dns_events,omitempty"`
	Queries        []*ArchivalDNSLookupEvent        `json:"queries,omitempty"`
	TCPConnect     []*ArchivalTCPConnect            `json:"tcp_connect,omitempty"`
	TLSHandshakes  []*ArchivalQUICTLSHandshakeEvent `json:"tls_handshakes,omitempty"`
	QUICHandshakes []*ArchivalQUICTLSHandshakeEvent `json:"quic_handshakes,omitempty"`
	Requests       []*ArchivalHTTPRoundTripEvent    `json:"requests,omitempty"`
}

// NewArchivalMeasurement converts a Measurement to ArchivalMeasurement.
func NewArchivalMeasurement(in *Measurement) *ArchivalMeasurement {
	out := &ArchivalMeasurement{
		NetworkEvents:  NewArchivalNetworkEventList(in.ReadWrite),
		DNSEvents:      NewArchivalDNSRoundTripEventList(in.DNSRoundTrip),
		Queries:        nil, // done below
		TCPConnect:     NewArchivalTCPConnectList(in.Connect),
		TLSHandshakes:  NewArchivalQUICTLSHandshakeEventList(in.TLSHandshake),
		QUICHandshakes: NewArchivalQUICTLSHandshakeEventList(in.QUICHandshake),
		Requests:       NewArchivalHTTPRoundTripEventList(in.HTTPRoundTrip),
	}
	out.Queries = append(out.Queries, NewArchivalDNSLookupEventList(in.LookupHost)...)
	out.Queries = append(out.Queries, NewArchivalDNSLookupEventList(in.LookupHTTPSSvc)...)
	return out
}
