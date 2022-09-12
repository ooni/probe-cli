package model

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"unicode/utf8"
)

//
// Archival format for individual measurement results
// such as TCP connect, TLS handshake, DNS lookup.
//
// These types end up inside the TestKeys field of an
// OONI measurement (see measurement.go).
//
// See https://github.com/ooni/spec/tree/master/data-formats.
//

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

// ArchivalMaybeBinaryData is a possibly binary string. We use this helper class
// to define a custom JSON encoder that allows us to choose the proper
// representation depending on whether the Value field is valid UTF-8 or not.
//
// See https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md#maybebinarydata
type ArchivalMaybeBinaryData struct {
	Value string
}

// MarshalJSON marshals a string-like to JSON following the OONI spec that
// says that UTF-8 content is represented as string and non-UTF-8 content is
// instead represented using `{"format":"base64","data":"..."}`.
func (hb ArchivalMaybeBinaryData) MarshalJSON() ([]byte, error) {
	if utf8.ValidString(hb.Value) {
		return json.Marshal(hb.Value)
	}
	er := make(map[string]string)
	er["format"] = "base64"
	er["data"] = base64.StdEncoding.EncodeToString([]byte(hb.Value))
	return json.Marshal(er)
}

// UnmarshalJSON is the opposite of MarshalJSON.
func (hb *ArchivalMaybeBinaryData) UnmarshalJSON(d []byte) error {
	if err := json.Unmarshal(d, &hb.Value); err == nil {
		return nil
	}
	er := make(map[string]string)
	if err := json.Unmarshal(d, &er); err != nil {
		return err
	}
	if v, ok := er["format"]; !ok || v != "base64" {
		return errors.New("missing or invalid format field")
	}
	if _, ok := er["data"]; !ok {
		return errors.New("missing data field")
	}
	b64, err := base64.StdEncoding.DecodeString(er["data"])
	if err != nil {
		return err
	}
	hb.Value = string(b64)
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
	Network            string                    `json:"network"`
	Address            string                    `json:"address"`
	CipherSuite        string                    `json:"cipher_suite"`
	Failure            *string                   `json:"failure"`
	SoError            *string                   `json:"so_error,omitempty"`
	NegotiatedProtocol string                    `json:"negotiated_protocol"`
	NoTLSVerify        bool                      `json:"no_tls_verify"`
	PeerCertificates   []ArchivalMaybeBinaryData `json:"peer_certificates"`
	ServerName         string                    `json:"server_name"`
	T0                 float64                   `json:"t0,omitempty"`
	T                  float64                   `json:"t"`
	Tags               []string                  `json:"tags"`
	TLSVersion         string                    `json:"tls_version"`
	TransactionID      int64                     `json:"transaction_id,omitempty"`
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
	TransactionID int64                `json:"transaction_id,omitempty"`
}

// ArchivalHTTPRequest contains an HTTP request.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type ArchivalHTTPRequest struct {
	Body            ArchivalHTTPBody                   `json:"body"`
	BodyIsTruncated bool                               `json:"body_is_truncated"`
	HeadersList     []ArchivalHTTPHeader               `json:"headers_list"`
	Headers         map[string]ArchivalMaybeBinaryData `json:"headers"`
	Method          string                             `json:"method"`
	Tor             ArchivalHTTPTor                    `json:"tor"`
	Transport       string                             `json:"x_transport"`
	URL             string                             `json:"url"`
}

// ArchivalHTTPResponse contains an HTTP response.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type ArchivalHTTPResponse struct {
	Body            ArchivalHTTPBody                   `json:"body"`
	BodyIsTruncated bool                               `json:"body_is_truncated"`
	Code            int64                              `json:"code"`
	HeadersList     []ArchivalHTTPHeader               `json:"headers_list"`
	Headers         map[string]ArchivalMaybeBinaryData `json:"headers"`

	// The following fields are not serialised but are useful to simplify
	// analysing the measurements in telegram, whatsapp, etc.
	Locations []string `json:"-"`
}

// ArchivalHTTPBody is an HTTP body. As an implementation note, this type must
// be an alias for the MaybeBinaryValue type, otherwise the specific serialisation
// mechanism implemented by MaybeBinaryValue is not working.
type ArchivalHTTPBody = ArchivalMaybeBinaryData

// ArchivalHTTPHeader is a single HTTP header.
type ArchivalHTTPHeader struct {
	Key   string
	Value ArchivalMaybeBinaryData
}

// MarshalJSON marshals a single HTTP header to a tuple where the first
// element is a string and the second element is maybe-binary data.
func (hh ArchivalHTTPHeader) MarshalJSON() ([]byte, error) {
	if utf8.ValidString(hh.Value.Value) {
		return json.Marshal([]string{hh.Key, hh.Value.Value})
	}
	value := make(map[string]string)
	value["format"] = "base64"
	value["data"] = base64.StdEncoding.EncodeToString([]byte(hh.Value.Value))
	return json.Marshal([]interface{}{hh.Key, value})
}

// UnmarshalJSON is the opposite of MarshalJSON.
func (hh *ArchivalHTTPHeader) UnmarshalJSON(d []byte) error {
	var pair []interface{}
	if err := json.Unmarshal(d, &pair); err != nil {
		return err
	}
	if len(pair) != 2 {
		return errors.New("unexpected pair length")
	}
	key, ok := pair[0].(string)
	if !ok {
		return errors.New("the key is not a string")
	}
	value, ok := pair[1].(string)
	if !ok {
		mapvalue, ok := pair[1].(map[string]interface{})
		if !ok {
			return errors.New("the value is neither a string nor a map[string]interface{}")
		}
		if _, ok := mapvalue["format"]; !ok {
			return errors.New("missing format")
		}
		if v, ok := mapvalue["format"].(string); !ok || v != "base64" {
			return errors.New("invalid format")
		}
		if _, ok := mapvalue["data"]; !ok {
			return errors.New("missing data field")
		}
		v, ok := mapvalue["data"].(string)
		if !ok {
			return errors.New("the data field is not a string")
		}
		b64, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return err
		}
		value = string(b64)
	}
	hh.Key, hh.Value = key, ArchivalMaybeBinaryData{Value: value}
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
