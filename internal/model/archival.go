package model

//
// Archival format for individual measurement results
// such as TCP connect, TLS handshake, DNS lookup.
//
// These types end up inside the TestKeys field of an
// OONI measurement (see measurement.go).
//
// See https://github.com/ooni/spec/tree/master/data-formats.
//

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"unicode/utf8"

	"github.com/ooni/probe-cli/v3/internal/scrubber"
)

//
// Data format extension specification
//

// ArchivalExtSpec describes a data format extension
type ArchivalExtSpec struct {
	Name string // extension name
	V    int64  // extension version
}

// AddTo adds the current ExtSpec to the specified measurement
func (spec ArchivalExtSpec) AddTo(m *Measurement) {
	if m.Extensions == nil {
		m.Extensions = make(map[string]int64)
	}
	m.Extensions[spec.Name] = spec.V
}

var (
	// ArchivalExtDNS is the version of df-002-dnst.md
	ArchivalExtDNS = ArchivalExtSpec{Name: "dnst", V: 0}

	// ArchivalExtNetevents is the version of df-008-netevents.md
	ArchivalExtNetevents = ArchivalExtSpec{Name: "netevents", V: 0}

	// ArchivalExtHTTP is the version of df-001-httpt.md
	ArchivalExtHTTP = ArchivalExtSpec{Name: "httpt", V: 0}

	// ArchivalExtTCPConnect is the version of df-005-tcpconnect.md
	ArchivalExtTCPConnect = ArchivalExtSpec{Name: "tcpconnect", V: 0}

	// ArchivalExtTLSHandshake is the version of df-006-tlshandshake.md
	ArchivalExtTLSHandshake = ArchivalExtSpec{Name: "tlshandshake", V: 0}

	// ArchivalExtTunnel is the version of df-009-tunnel.md
	ArchivalExtTunnel = ArchivalExtSpec{Name: "tunnel", V: 0}
)

//
// Base types
//

// ArchivalBinaryData is a wrapper for bytes that serializes the enclosed
// data using the specific ooni/spec data format for binary data.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md#maybebinarydata.
type ArchivalBinaryData []byte

// archivalBinaryDataRepr is the wire representation of binary data according to
// https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md#maybebinarydata.
type archivalBinaryDataRepr struct {
	Data   []byte `json:"data"`
	Format string `json:"format"`
}

var (
	_ json.Marshaler   = ArchivalBinaryData{}
	_ json.Unmarshaler = &ArchivalBinaryData{}
)

// MarshalJSON implements json.Marshaler.
func (value ArchivalBinaryData) MarshalJSON() ([]byte, error) {
	// special case: we need to marshal the empty data as the null value
	if len(value) <= 0 {
		return json.Marshal(nil)
	}

	// construct and serialize the OONI representation
	repr := &archivalBinaryDataRepr{Format: "base64", Data: value}
	return json.Marshal(repr)
}

// ErrInvalidBinaryDataFormat is the format returned when marshaling and
// unmarshaling binary data and the value of "format" is unknown.
var ErrInvalidBinaryDataFormat = errors.New("model: invalid binary data format")

// UnmarshalJSON implements json.Unmarshaler.
func (value *ArchivalBinaryData) UnmarshalJSON(raw []byte) error {
	// handle the case where input is a literal null
	if bytes.Equal(raw, []byte("null")) {
		*value = nil
		return nil
	}

	// attempt to unmarshal into the archival representation
	var repr archivalBinaryDataRepr
	if err := json.Unmarshal(raw, &repr); err != nil {
		return err
	}

	// make sure the data format is "base64"
	if repr.Format != "base64" {
		return fmt.Errorf("%w: '%s'", ErrInvalidBinaryDataFormat, repr.Format)
	}

	// we're good because Go uses base64 for []byte automatically
	*value = repr.Data
	return nil
}

// ArchivalScrubbedMaybeBinaryString is a possibly-binary string. When the string is valid UTF-8
// we serialize it as itself. Otherwise, we use the binary data format defined by
// https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md#maybebinarydata
//
// As the name implies, the data contained by this type is scrubbed to remove IPv4 and IPv6
// addresses and endpoints during JSON serialization, to make it less likely that OONI leaks
// IP addresses in textual or binary fields such as HTTP headers and bodies.
type ArchivalScrubbedMaybeBinaryString string

var (
	_ json.Marshaler   = ArchivalScrubbedMaybeBinaryString("")
	_ json.Unmarshaler = (func() *ArchivalScrubbedMaybeBinaryString { return nil }())
)

// MarshalJSON implements json.Marshaler.
func (value ArchivalScrubbedMaybeBinaryString) MarshalJSON() ([]byte, error) {
	// convert value to a string
	str := string(value)

	// make sure we get rid of IPv4 and IPv6 addresses and endpoints
	str = scrubber.ScrubString(str)

	// if we can serialize as UTF-8 string, do that
	if utf8.ValidString(str) {
		return json.Marshal(str)
	}

	// otherwise fallback to the serialization of ArchivalBinaryData
	return json.Marshal(ArchivalBinaryData(str))
}

// UnmarshalJSON implements json.Unmarshaler.
func (value *ArchivalScrubbedMaybeBinaryString) UnmarshalJSON(rawData []byte) error {
	// first attempt to decode as a string
	var s string
	if err := json.Unmarshal(rawData, &s); err == nil {
		*value = ArchivalScrubbedMaybeBinaryString(s)
		return nil
	}

	// then attempt to decode as ArchivalBinaryData
	var d ArchivalBinaryData
	if err := json.Unmarshal(rawData, &d); err != nil {
		return err
	}
	*value = ArchivalScrubbedMaybeBinaryString(d)
	return nil
}

//
// DNS lookup
//

// ArchivalDNSLookupResult is the result of a DNS lookup.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-002-dnst.md.
type ArchivalDNSLookupResult struct {
	Answers          []ArchivalDNSAnswer `json:"answers"`
	Engine           string              `json:"engine"`
	Failure          *string             `json:"failure"`
	GetaddrinfoError int64               `json:"getaddrinfo_error,omitempty"`
	Hostname         string              `json:"hostname"`
	QueryType        string              `json:"query_type"`
	RawResponse      []byte              `json:"raw_response,omitempty"`
	Rcode            int64               `json:"rcode,omitempty"`
	ResolverHostname *string             `json:"resolver_hostname"`
	ResolverPort     *string             `json:"resolver_port"`
	ResolverAddress  string              `json:"resolver_address"`
	T0               float64             `json:"t0,omitempty"`
	T                float64             `json:"t"`
	Tags             []string            `json:"tags"`
	TransactionID    int64               `json:"transaction_id,omitempty"`
}

// ArchivalDNSAnswer is a DNS answer.
type ArchivalDNSAnswer struct {
	ASN        int64   `json:"asn,omitempty"`
	ASOrgName  string  `json:"as_org_name,omitempty"`
	AnswerType string  `json:"answer_type"`
	Hostname   string  `json:"hostname,omitempty"`
	IPv4       string  `json:"ipv4,omitempty"`
	IPv6       string  `json:"ipv6,omitempty"`
	TTL        *uint32 `json:"ttl"`
}

//
// TCP connect
//

// ArchivalTCPConnectResult contains the result of a TCP connect.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-005-tcpconnect.md.
type ArchivalTCPConnectResult struct {
	IP            string                   `json:"ip"`
	Port          int                      `json:"port"`
	Status        ArchivalTCPConnectStatus `json:"status"`
	T0            float64                  `json:"t0,omitempty"`
	T             float64                  `json:"t"`
	Tags          []string                 `json:"tags"`
	TransactionID int64                    `json:"transaction_id,omitempty"`
}

// ArchivalTCPConnectStatus is the status of ArchivalTCPConnectResult.
type ArchivalTCPConnectStatus struct {
	Blocked *bool   `json:"blocked,omitempty"`
	Failure *string `json:"failure"`
	Success bool    `json:"success"`
}

//
// TLS or QUIC handshake
//

// ArchivalTLSOrQUICHandshakeResult is the result of a TLS or QUIC handshake.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-006-tlshandshake.md
type ArchivalTLSOrQUICHandshakeResult struct {
	Network            string               `json:"network"`
	Address            string               `json:"address"`
	CipherSuite        string               `json:"cipher_suite"`
	Failure            *string              `json:"failure"`
	SoError            *string              `json:"so_error,omitempty"`
	NegotiatedProtocol string               `json:"negotiated_protocol"`
	NoTLSVerify        bool                 `json:"no_tls_verify"`
	PeerCertificates   []ArchivalBinaryData `json:"peer_certificates"`
	ServerName         string               `json:"server_name"`
	T0                 float64              `json:"t0,omitempty"`
	T                  float64              `json:"t"`
	Tags               []string             `json:"tags"`
	TLSVersion         string               `json:"tls_version"`
	TransactionID      int64                `json:"transaction_id,omitempty"`
}

//
// HTTP
//

// ArchivalHTTPRequestResult is the result of sending an HTTP request.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md.
type ArchivalHTTPRequestResult struct {
	Network       string               `json:"network,omitempty"`
	Address       string               `json:"address,omitempty"`
	ALPN          string               `json:"alpn,omitempty"`
	Failure       *string              `json:"failure"`
	Request       ArchivalHTTPRequest  `json:"request"`
	Response      ArchivalHTTPResponse `json:"response"`
	T0            float64              `json:"t0,omitempty"`
	T             float64              `json:"t"`
	Tags          []string             `json:"tags"`
	TransactionID int64                `json:"transaction_id,omitempty"`
}

// ArchivalHTTPRequest contains an HTTP request.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type ArchivalHTTPRequest struct {
	Body            ArchivalScrubbedMaybeBinaryString            `json:"body"`
	BodyIsTruncated bool                                         `json:"body_is_truncated"`
	HeadersList     []ArchivalHTTPHeader                         `json:"headers_list"`
	Headers         map[string]ArchivalScrubbedMaybeBinaryString `json:"headers"`
	Method          string                                       `json:"method"`
	Tor             ArchivalHTTPTor                              `json:"tor"`
	Transport       string                                       `json:"x_transport"`
	URL             string                                       `json:"url"`
}

// ArchivalHTTPResponse contains an HTTP response.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type ArchivalHTTPResponse struct {
	Body            ArchivalScrubbedMaybeBinaryString            `json:"body"`
	BodyIsTruncated bool                                         `json:"body_is_truncated"`
	Code            int64                                        `json:"code"`
	HeadersList     []ArchivalHTTPHeader                         `json:"headers_list"`
	Headers         map[string]ArchivalScrubbedMaybeBinaryString `json:"headers"`

	// The following fields are not serialised but are useful to simplify
	// analysing the measurements in telegram, whatsapp, etc.
	Locations []string `json:"-"`
}

// ArchivalNewHTTPHeadersList constructs a new ArchivalHTTPHeader list given HTTP headers.
func ArchivalNewHTTPHeadersList(source http.Header) (out []ArchivalHTTPHeader) {
	out = []ArchivalHTTPHeader{}

	// obtain the header keys
	keys := []string{}
	for key := range source {
		keys = append(keys, key)
	}

	// ensure the output is consistent, which helps with testing;
	// for an example of why we need to sort headers, see
	// https://github.com/ooni/probe-engine/pull/751/checks?check_run_id=853562310
	sort.Strings(keys)

	// insert into the output list
	for _, key := range keys {
		for _, value := range source[key] {
			out = append(out, ArchivalHTTPHeader{
				ArchivalScrubbedMaybeBinaryString(key),
				ArchivalScrubbedMaybeBinaryString(value),
			})
		}
	}
	return
}

// ArchivalNewHTTPHeadersMap creates a map representation of HTTP headers
func ArchivalNewHTTPHeadersMap(header http.Header) (out map[string]ArchivalScrubbedMaybeBinaryString) {
	out = make(map[string]ArchivalScrubbedMaybeBinaryString)
	for key, values := range header {
		for _, value := range values {
			out[key] = ArchivalScrubbedMaybeBinaryString(value)
			break // just the first header
		}
	}
	return
}

// ArchivalHTTPHeader is a single HTTP header.
type ArchivalHTTPHeader [2]ArchivalScrubbedMaybeBinaryString

// errCannotParseArchivalHTTPHeader indicates that we cannot parse an ArchivalHTTPHeader.
var errCannotParseArchivalHTTPHeader = errors.New("invalid ArchivalHTTPHeader")

// UnmarshalJSON implements json.Unmarshaler.
func (ahh *ArchivalHTTPHeader) UnmarshalJSON(data []byte) error {
	var helper []ArchivalScrubbedMaybeBinaryString
	if err := json.Unmarshal(data, &helper); err != nil {
		return err
	}
	if len(helper) != 2 {
		return fmt.Errorf("%w: expected 2 elements, got %d", errCannotParseArchivalHTTPHeader, len(helper))
	}
	(*ahh)[0] = helper[0]
	(*ahh)[1] = helper[1]
	return nil
}

// ArchivalHTTPTor contains Tor information.
type ArchivalHTTPTor struct {
	ExitIP   *string `json:"exit_ip"`
	ExitName *string `json:"exit_name"`
	IsTor    bool    `json:"is_tor"`
}

//
// NetworkEvent
//

// ArchivalNetworkEvent is a network event. It contains all the possible fields
// and most fields are optional. They are only added when it makes sense
// for them to be there _and_ we have data to show.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-008-netevents.md.
type ArchivalNetworkEvent struct {
	Address       string   `json:"address,omitempty"`
	Failure       *string  `json:"failure"`
	NumBytes      int64    `json:"num_bytes,omitempty"`
	Operation     string   `json:"operation"`
	Proto         string   `json:"proto,omitempty"`
	T0            float64  `json:"t0,omitempty"`
	T             float64  `json:"t"`
	TransactionID int64    `json:"transaction_id,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}

//
// OpenVPN
//

// ArchivalOpenVPNHandshakeResult contains the result of a OpenVPN handshake.
type ArchivalOpenVPNHandshakeResult struct {
	BootstrapTime  float64                `json:"bootstrap_time,omitempty"`
	Endpoint       string                 `json:"endpoint"`
	Failure        *string                `json:"failure"`
	IP             string                 `json:"ip"`
	Port           int                    `json:"port"`
	Transport      string                 `json:"transport"`
	Provider       string                 `json:"provider"`
	OpenVPNOptions ArchivalOpenVPNOptions `json:"openvpn_options"`
	T0             float64                `json:"t0,omitempty"`
	T              float64                `json:"t"`
	Tags           []string               `json:"tags"`
	TransactionID  int64                  `json:"transaction_id,omitempty"`
}

// ArchivalOpenVPNOptions is a subset of [vpnconfig.OpenVPNOptions] that we want to include
// in the archived result.
type ArchivalOpenVPNOptions struct {
	Auth        string `json:"auth,omitempty"`
	Cipher      string `json:"cipher,omitempty"`
	Compression string `json:"compression,omitempty"`
}
