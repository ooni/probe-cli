package measurex

import (
	"log"
	"net/http"
	"strings"
)

//
// Archival
//
// This file defines helpers to serialize to the OONI data format. Some of
// our data structure are already pretty close to the desired format, while
// other are more flat, which makes processing simpler. So, when we need
// help we use routines from this file to serialize correctly.
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
// HTTPRoundTrip
//

// ArchivalHeadersList is a list of HTTP headers.
type ArchivalHeadersList [][]string

// Get searches for the first header with the named key
// and returns it. If not found, returns an empty string.
func (headers ArchivalHeadersList) Get(key string) string {
	key = strings.ToLower(key)
	for _, entry := range headers {
		if len(entry) != 2 {
			log.Printf("headers: malformed header: %+v", entry)
			continue
		}
		headerKey, headerValue := entry[0], entry[1]
		if strings.ToLower(headerKey) == key {
			return headerValue
		}
	}
	return ""
}

// NewArchivalHeadersList builds a new HeadersList from http.Header.
func NewArchivalHeadersList(in http.Header) (out ArchivalHeadersList) {
	for k, vv := range in {
		for _, v := range vv {
			out = append(out, []string{k, v})
		}
	}
	return
}

//
// TLSCerts
//

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

//
// DNS LookupHost and LookupHTTPSSvc
//

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

// NewArchivalLookupHostList converts a []*LookupHostEvent
// to the corresponding archival format.
func NewArchivalLookupHostList(in ...*LookupHostEvent) (out []*ArchivalDNSLookup) {
	for _, ev := range in {
		out = append(out, NewArchivalLookupHost(ev, "A"))
		out = append(out, NewArchivalLookupHost(ev, "AAAA"))
	}
	return
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
		Finished:  in.Finished,
		Started:   in.Started,
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
		Finished:  in.Finished,
		Started:   in.Started,
		Oddity:    in.Oddity,
	}
}

// NewArchivalLookupHTTPSSvcList converts a []*LookupHTTPSSvcEvent
// to the corresponding archival format.
func NewArchivalLookupHTTPSSvcList(in ...*LookupHTTPSSvcEvent) (out []*ArchivalDNSLookup) {
	for _, ev := range in {
		out = append(out, NewArchivalLookupHTTPSSvc(ev))
	}
	return
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
