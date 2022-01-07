// Package archival contains data formats used for archival.
//
// See https://github.com/ooni/spec.
package archival

import (
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ExtSpec describes a data format extension
type ExtSpec struct {
	Name string // extension name
	V    int64  // extension version
}

// AddTo adds the current ExtSpec to the specified measurement
func (spec ExtSpec) AddTo(m *model.Measurement) {
	if m.Extensions == nil {
		m.Extensions = make(map[string]int64)
	}
	m.Extensions[spec.Name] = spec.V
}

var (
	// ExtDNS is the version of df-002-dnst.md
	ExtDNS = ExtSpec{Name: "dnst", V: 0}

	// ExtNetevents is the version of df-008-netevents.md
	ExtNetevents = ExtSpec{Name: "netevents", V: 0}

	// ExtHTTP is the version of df-001-httpt.md
	ExtHTTP = ExtSpec{Name: "httpt", V: 0}

	// ExtTCPConnect is the version of df-005-tcpconnect.md
	ExtTCPConnect = ExtSpec{Name: "tcpconnect", V: 0}

	// ExtTLSHandshake is the version of df-006-tlshandshake.md
	ExtTLSHandshake = ExtSpec{Name: "tlshandshake", V: 0}

	// ExtTunnel is the version of df-009-tunnel.md
	ExtTunnel = ExtSpec{Name: "tunnel", V: 0}
)

// TCPConnectStatus contains the TCP connect status.
//
// The Blocked field breaks the separation between measurement and analysis
// we have been enforcing for quite some time now. It is a legacy from the
// Web Connectivity experiment and it should be here because of that.
type TCPConnectStatus struct {
	Blocked *bool   `json:"blocked,omitempty"` // Web Connectivity only
	Failure *string `json:"failure"`
	Success bool    `json:"success"`
}

// TCPConnectEntry contains one of the entries that are part
// of the "tcp_connect" key of a OONI report.
type TCPConnectEntry struct {
	IP     string           `json:"ip"`
	Port   int              `json:"port"`
	Status TCPConnectStatus `json:"status"`
	T      float64          `json:"t"`
}

// NewTCPConnectList creates a new TCPConnectList
func NewTCPConnectList(begin time.Time, events []trace.Event) []TCPConnectEntry {
	var out []TCPConnectEntry
	for _, event := range events {
		if event.Name != netxlite.ConnectOperation {
			continue
		}
		if event.Proto != "tcp" {
			continue
		}
		// We assume Go is passing us legit data structures
		ip, sport, _ := net.SplitHostPort(event.Address)
		iport, _ := strconv.Atoi(sport)
		out = append(out, TCPConnectEntry{
			IP:   ip,
			Port: iport,
			Status: TCPConnectStatus{
				Failure: NewFailure(event.Err),
				Success: event.Err == nil,
			},
			T: event.Time.Sub(begin).Seconds(),
		})
	}
	return out
}

// NewFailure creates a failure nullable string from the given error
func NewFailure(err error) *string {
	if err == nil {
		return nil
	}
	// The following code guarantees that the error is always wrapped even
	// when we could not actually hit our code that does the wrapping. A case
	// in which this happen is with context deadline for HTTP.
	err = netxlite.NewTopLevelGenericErrWrapper(err)
	errWrapper := err.(*netxlite.ErrWrapper)
	s := errWrapper.Failure
	if s == "" {
		s = "unknown_failure: errWrapper.Failure is empty"
	}
	return &s
}

// NewFailedOperation creates a failed operation string from the given error.
func NewFailedOperation(err error) *string {
	if err == nil {
		return nil
	}
	var (
		errWrapper *netxlite.ErrWrapper
		s          = netxlite.UnknownOperation
	)
	if errors.As(err, &errWrapper) && errWrapper.Operation != "" {
		s = errWrapper.Operation
	}
	return &s
}

// HTTPTor contains Tor information
type HTTPTor struct {
	ExitIP   *string `json:"exit_ip"`
	ExitName *string `json:"exit_name"`
	IsTor    bool    `json:"is_tor"`
}

// MaybeBinaryValue is a possibly binary string. We use this helper class
// to define a custom JSON encoder that allows us to choose the proper
// representation depending on whether the Value field is valid UTF-8 or not.
type MaybeBinaryValue struct {
	Value string
}

// MarshalJSON marshals a string-like to JSON following the OONI spec that
// says that UTF-8 content is represened as string and non-UTF-8 content is
// instead represented using `{"format":"base64","data":"..."}`.
func (hb MaybeBinaryValue) MarshalJSON() ([]byte, error) {
	if utf8.ValidString(hb.Value) {
		return json.Marshal(hb.Value)
	}
	er := make(map[string]string)
	er["format"] = "base64"
	er["data"] = base64.StdEncoding.EncodeToString([]byte(hb.Value))
	return json.Marshal(er)
}

// UnmarshalJSON is the opposite of MarshalJSON.
func (hb *MaybeBinaryValue) UnmarshalJSON(d []byte) error {
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

// HTTPBody is an HTTP body. As an implementation note, this type must be
// an alias for the MaybeBinaryValue type, otherwise the specific serialisation
// mechanism implemented by MaybeBinaryValue is not working.
type HTTPBody = MaybeBinaryValue

// HTTPHeader is a single HTTP header.
type HTTPHeader struct {
	Key   string
	Value MaybeBinaryValue
}

// MarshalJSON marshals a single HTTP header to a tuple where the first
// element is a string and the second element is maybe-binary data.
func (hh HTTPHeader) MarshalJSON() ([]byte, error) {
	if utf8.ValidString(hh.Value.Value) {
		return json.Marshal([]string{hh.Key, hh.Value.Value})
	}
	value := make(map[string]string)
	value["format"] = "base64"
	value["data"] = base64.StdEncoding.EncodeToString([]byte(hh.Value.Value))
	return json.Marshal([]interface{}{hh.Key, value})
}

// UnmarshalJSON is the opposite of MarshalJSON.
func (hh *HTTPHeader) UnmarshalJSON(d []byte) error {
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
	hh.Key, hh.Value = key, MaybeBinaryValue{Value: value}
	return nil
}

// HTTPRequest contains an HTTP request.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type HTTPRequest struct {
	Body            HTTPBody                    `json:"body"`
	BodyIsTruncated bool                        `json:"body_is_truncated"`
	HeadersList     []HTTPHeader                `json:"headers_list"`
	Headers         map[string]MaybeBinaryValue `json:"headers"`
	Method          string                      `json:"method"`
	Tor             HTTPTor                     `json:"tor"`
	Transport       string                      `json:"x_transport"`
	URL             string                      `json:"url"`
}

// HTTPResponse contains an HTTP response.
//
// Headers are a map in Web Connectivity data format but
// we have added support for a list since January 2020.
type HTTPResponse struct {
	Body            HTTPBody                    `json:"body"`
	BodyIsTruncated bool                        `json:"body_is_truncated"`
	Code            int64                       `json:"code"`
	HeadersList     []HTTPHeader                `json:"headers_list"`
	Headers         map[string]MaybeBinaryValue `json:"headers"`

	// The following fields are not serialised but are useful to simplify
	// analysing the measurements in telegram, whatsapp, etc.
	Locations []string `json:"-"`
}

// RequestEntry is one of the entries that are part of
// the "requests" key of a OONI report.
type RequestEntry struct {
	Failure  *string      `json:"failure"`
	Request  HTTPRequest  `json:"request"`
	Response HTTPResponse `json:"response"`
	T        float64      `json:"t"`
}

func addheaders(
	source http.Header,
	destList *[]HTTPHeader,
	destMap *map[string]MaybeBinaryValue,
) {
	for key, values := range source {
		for index, value := range values {
			value := MaybeBinaryValue{Value: value}
			// With the map representation we can only represent a single
			// value for every key. Hence the list representation.
			if index == 0 {
				(*destMap)[key] = value
			}
			*destList = append(*destList, HTTPHeader{
				Key:   key,
				Value: value,
			})
		}
	}
	sort.Slice(*destList, func(i, j int) bool {
		return (*destList)[i].Key < (*destList)[j].Key
	})
}

// NewRequestList returns the list for "requests"
func NewRequestList(begin time.Time, events []trace.Event) []RequestEntry {
	// OONI wants the last request to appear first
	var out []RequestEntry
	tmp := newRequestList(begin, events)
	for i := len(tmp) - 1; i >= 0; i-- {
		out = append(out, tmp[i])
	}
	return out
}

func newRequestList(begin time.Time, events []trace.Event) []RequestEntry {
	var (
		out   []RequestEntry
		entry RequestEntry
	)
	for _, ev := range events {
		switch ev.Name {
		case "http_transaction_start":
			entry = RequestEntry{}
			entry.T = ev.Time.Sub(begin).Seconds()
		case "http_request_body_snapshot":
			entry.Request.Body.Value = string(ev.Data)
			entry.Request.BodyIsTruncated = ev.DataIsTruncated
		case "http_request_metadata":
			entry.Request.Headers = make(map[string]MaybeBinaryValue)
			addheaders(
				ev.HTTPHeaders, &entry.Request.HeadersList, &entry.Request.Headers)
			entry.Request.Method = ev.HTTPMethod
			entry.Request.URL = ev.HTTPURL
			entry.Request.Transport = ev.Transport
		case "http_response_metadata":
			entry.Response.Headers = make(map[string]MaybeBinaryValue)
			addheaders(
				ev.HTTPHeaders, &entry.Response.HeadersList, &entry.Response.Headers)
			entry.Response.Code = int64(ev.HTTPStatusCode)
			entry.Response.Locations = ev.HTTPHeaders.Values("Location")
		case "http_response_body_snapshot":
			entry.Response.Body.Value = string(ev.Data)
			entry.Response.BodyIsTruncated = ev.DataIsTruncated
		case "http_transaction_done":
			entry.Failure = NewFailure(ev.Err)
			out = append(out, entry)
		}
	}
	return out
}

// DNSAnswerEntry is the answer to a DNS query
type DNSAnswerEntry struct {
	ASN        int64   `json:"asn,omitempty"`
	ASOrgName  string  `json:"as_org_name,omitempty"`
	AnswerType string  `json:"answer_type"`
	Hostname   string  `json:"hostname,omitempty"`
	IPv4       string  `json:"ipv4,omitempty"`
	IPv6       string  `json:"ipv6,omitempty"`
	TTL        *uint32 `json:"ttl"`
}

// DNSQueryEntry is a DNS query with possibly an answer
type DNSQueryEntry struct {
	Answers          []DNSAnswerEntry `json:"answers"`
	Engine           string           `json:"engine"`
	Failure          *string          `json:"failure"`
	Hostname         string           `json:"hostname"`
	QueryType        string           `json:"query_type"`
	ResolverHostname *string          `json:"resolver_hostname"`
	ResolverPort     *string          `json:"resolver_port"`
	ResolverAddress  string           `json:"resolver_address"`
	T                float64          `json:"t"`
}

type dnsQueryType string

// NewDNSQueriesList returns a list of DNS queries.
func NewDNSQueriesList(begin time.Time, events []trace.Event) []DNSQueryEntry {
	// TODO(bassosimone): add support for CNAME lookups.
	var out []DNSQueryEntry
	for _, ev := range events {
		if ev.Name != "resolve_done" {
			continue
		}
		for _, qtype := range []dnsQueryType{"A", "AAAA"} {
			entry := qtype.makequeryentry(begin, ev)
			for _, addr := range ev.Addresses {
				if qtype.ipoftype(addr) {
					entry.Answers = append(
						entry.Answers, qtype.makeanswerentry(addr))
				}
			}
			if len(entry.Answers) <= 0 && ev.Err == nil {
				// This allows us to skip cases where the server does not have
				// an IPv6 address but has an IPv4 address. Instead, when we
				// receive an error, we want to track its existence. The main
				// issue here is that we are cheating, because we are creating
				// entries representing queries, but we don't know what the
				// resolver actually did, especially the system resolver. So,
				// this output is just our best guess.
				continue
			}
			out = append(out, entry)
		}
	}
	return out
}

func (qtype dnsQueryType) ipoftype(addr string) bool {
	switch qtype {
	case "A":
		return !strings.Contains(addr, ":")
	case "AAAA":
		return strings.Contains(addr, ":")
	}
	return false
}

func (qtype dnsQueryType) makeanswerentry(addr string) DNSAnswerEntry {
	answer := DNSAnswerEntry{AnswerType: string(qtype)}
	asn, org, _ := geolocate.LookupASN(addr)
	answer.ASN = int64(asn)
	answer.ASOrgName = org
	switch qtype {
	case "A":
		answer.IPv4 = addr
	case "AAAA":
		answer.IPv6 = addr
	}
	return answer
}

func (qtype dnsQueryType) makequeryentry(begin time.Time, ev trace.Event) DNSQueryEntry {
	return DNSQueryEntry{
		Engine:          ev.Proto,
		Failure:         NewFailure(ev.Err),
		Hostname:        ev.Hostname,
		QueryType:       string(qtype),
		ResolverAddress: ev.Address,
		T:               ev.Time.Sub(begin).Seconds(),
	}
}

// NetworkEvent is a network event. It contains all the possible fields
// and most fields are optional. They are only added when it makes sense
// for them to be there _and_ we have data to show.
type NetworkEvent struct {
	Address   string   `json:"address,omitempty"`
	Failure   *string  `json:"failure"`
	NumBytes  int64    `json:"num_bytes,omitempty"`
	Operation string   `json:"operation"`
	Proto     string   `json:"proto,omitempty"`
	T         float64  `json:"t"`
	Tags      []string `json:"tags,omitempty"`
}

// NewNetworkEventsList returns a list of DNS queries.
func NewNetworkEventsList(begin time.Time, events []trace.Event) []NetworkEvent {
	var out []NetworkEvent
	for _, ev := range events {
		if ev.Name == netxlite.ConnectOperation {
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   NewFailure(ev.Err),
				Operation: ev.Name,
				Proto:     ev.Proto,
				T:         ev.Time.Sub(begin).Seconds(),
			})
			continue
		}
		if ev.Name == netxlite.ReadOperation {
			out = append(out, NetworkEvent{
				Failure:   NewFailure(ev.Err),
				Operation: ev.Name,
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
			continue
		}
		if ev.Name == netxlite.WriteOperation {
			out = append(out, NetworkEvent{
				Failure:   NewFailure(ev.Err),
				Operation: ev.Name,
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
			continue
		}
		if ev.Name == netxlite.ReadFromOperation {
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   NewFailure(ev.Err),
				Operation: ev.Name,
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
			continue
		}
		if ev.Name == netxlite.WriteToOperation {
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   NewFailure(ev.Err),
				Operation: ev.Name,
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
			continue
		}
		out = append(out, NetworkEvent{
			Failure:   NewFailure(ev.Err),
			Operation: ev.Name,
			T:         ev.Time.Sub(begin).Seconds(),
		})
	}
	return out
}

// TLSHandshake contains TLS handshake data
type TLSHandshake struct {
	CipherSuite        string             `json:"cipher_suite"`
	Failure            *string            `json:"failure"`
	NegotiatedProtocol string             `json:"negotiated_protocol"`
	NoTLSVerify        bool               `json:"no_tls_verify"`
	PeerCertificates   []MaybeBinaryValue `json:"peer_certificates"`
	ServerName         string             `json:"server_name"`
	T                  float64            `json:"t"`
	Tags               []string           `json:"tags"`
	TLSVersion         string             `json:"tls_version"`
}

// NewTLSHandshakesList creates a new TLSHandshakesList
func NewTLSHandshakesList(begin time.Time, events []trace.Event) []TLSHandshake {
	var out []TLSHandshake
	for _, ev := range events {
		if !strings.Contains(ev.Name, "_handshake_done") {
			continue
		}
		out = append(out, TLSHandshake{
			CipherSuite:        ev.TLSCipherSuite,
			Failure:            NewFailure(ev.Err),
			NegotiatedProtocol: ev.TLSNegotiatedProto,
			NoTLSVerify:        ev.NoTLSVerify,
			PeerCertificates:   makePeerCerts(ev.TLSPeerCerts),
			ServerName:         ev.TLSServerName,
			T:                  ev.Time.Sub(begin).Seconds(),
			TLSVersion:         ev.TLSVersion,
		})
	}
	return out
}

func makePeerCerts(in []*x509.Certificate) (out []MaybeBinaryValue) {
	for _, e := range in {
		out = append(out, MaybeBinaryValue{Value: string(e.Raw)})
	}
	return
}
