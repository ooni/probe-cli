package tracex

//
// Code to generate the OONI archival data format from events
//

import (
	"errors"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Compatibility types. Most experiments still use these names.
type (
	ExtSpec          = model.ArchivalExtSpec
	TCPConnectEntry  = model.ArchivalTCPConnectResult
	TCPConnectStatus = model.ArchivalTCPConnectStatus
	MaybeBinaryValue = model.ArchivalMaybeBinaryData
	DNSQueryEntry    = model.ArchivalDNSLookupResult
	DNSAnswerEntry   = model.ArchivalDNSAnswer
	TLSHandshake     = model.ArchivalTLSOrQUICHandshakeResult
	HTTPBody         = model.ArchivalHTTPBody
	HTTPHeader       = model.ArchivalHTTPHeader
	RequestEntry     = model.ArchivalHTTPRequestResult
	HTTPRequest      = model.ArchivalHTTPRequest
	HTTPResponse     = model.ArchivalHTTPResponse
	NetworkEvent     = model.ArchivalNetworkEvent
)

// Compatibility variables. Most experiments still use these names.
var (
	ExtDNS          = model.ArchivalExtDNS
	ExtNetevents    = model.ArchivalExtNetevents
	ExtHTTP         = model.ArchivalExtHTTP
	ExtTCPConnect   = model.ArchivalExtTCPConnect
	ExtTLSHandshake = model.ArchivalExtTLSHandshake
	ExtTunnel       = model.ArchivalExtTunnel
)

// NewTCPConnectList creates a new TCPConnectList
func NewTCPConnectList(begin time.Time, events []Event) (out []TCPConnectEntry) {
	for _, wrapper := range events {
		if _, ok := wrapper.(*EventConnectOperation); !ok {
			continue
		}
		event := wrapper.Value()
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
				Blocked: nil, // only used by Web Connectivity
				Failure: event.Err.ToFailure(),
				Success: event.Err.IsNil(),
			},
			T: event.Time.Sub(begin).Seconds(),
		})
	}
	return
}

// TODO(bassosimone): can we use measurexlite.NewFailure here? Do we need to have
// a string (as opposed to *string) representation of failures here?

// NewFailure creates a failure nullable string from the given error. This function
// is equivalent to NewFailureStr(err).ToFailure().
func NewFailure(err error) *string {
	return NewFailureStr(err).ToFailure()
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

// httpAddHeaders adds the headers inside source into destList and destMap.
func httpAddHeaders(source http.Header, destList *[]HTTPHeader,
	destMap *map[string]MaybeBinaryValue) {
	*destList = []HTTPHeader{}
	*destMap = make(map[string]model.ArchivalMaybeBinaryData)
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
	// Sorting helps with unit testing (map keys are unordered)
	sort.Slice(*destList, func(i, j int) bool {
		return (*destList)[i].Key < (*destList)[j].Key
	})
}

// NewRequestList returns the list for "requests"
func NewRequestList(begin time.Time, events []Event) (out []RequestEntry) {
	// OONI wants the last request to appear first
	tmp := newRequestList(begin, events)
	for i := len(tmp) - 1; i >= 0; i-- {
		out = append(out, tmp[i])
	}
	return
}

func newRequestList(begin time.Time, events []Event) (out []RequestEntry) {
	for _, wrapper := range events {
		ev := wrapper.Value()
		switch wrapper.(type) {
		case *EventHTTPTransactionDone:
			entry := RequestEntry{}
			entry.T = ev.Time.Sub(begin).Seconds()
			httpAddHeaders(
				ev.HTTPRequestHeaders, &entry.Request.HeadersList, &entry.Request.Headers)
			entry.Request.Method = ev.HTTPMethod
			entry.Request.URL = ev.HTTPURL
			entry.Request.Transport = ev.Transport
			httpAddHeaders(
				ev.HTTPResponseHeaders, &entry.Response.HeadersList, &entry.Response.Headers)
			entry.Response.Code = int64(ev.HTTPStatusCode)
			entry.Response.Locations = ev.HTTPResponseHeaders.Values("Location")
			entry.Response.Body.Value = string(ev.HTTPResponseBody)
			entry.Response.BodyIsTruncated = ev.HTTPResponseBodyIsTruncated
			entry.Failure = ev.Err.ToFailure()
			out = append(out, entry)
		}
	}
	return
}

type dnsQueryType string

// NewDNSQueriesList returns a list of DNS queries.
func NewDNSQueriesList(begin time.Time, events []Event) (out []DNSQueryEntry) {
	// TODO(bassosimone): add support for CNAME lookups.
	for _, wrapper := range events {
		if _, ok := wrapper.(*EventResolveDone); !ok {
			continue
		}
		ev := wrapper.Value()
		for _, qtype := range []dnsQueryType{"A", "AAAA"} {
			entry := qtype.makeQueryEntry(begin, ev)
			for _, addr := range ev.Addresses {
				if qtype.ipOfType(addr) {
					entry.Answers = append(
						entry.Answers, qtype.makeAnswerEntry(addr))
				}
			}
			if len(entry.Answers) <= 0 && ev.Err.IsNil() {
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
	return
}

func (qtype dnsQueryType) ipOfType(addr string) bool {
	switch qtype {
	case "A":
		return !strings.Contains(addr, ":")
	case "AAAA":
		return strings.Contains(addr, ":")
	}
	return false
}

func (qtype dnsQueryType) makeAnswerEntry(addr string) DNSAnswerEntry {
	answer := DNSAnswerEntry{AnswerType: string(qtype)}
	// Figuring out the ASN and the org here is not just a service to whoever
	// is reading a JSON: Web Connectivity also depends on it!
	asn, org, _ := geoipx.LookupASN(addr)
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

func (qtype dnsQueryType) makeQueryEntry(begin time.Time, ev *EventValue) DNSQueryEntry {
	return DNSQueryEntry{
		Engine:          ev.Proto,
		Failure:         ev.Err.ToFailure(),
		Hostname:        ev.Hostname,
		QueryType:       string(qtype),
		ResolverAddress: ev.Address,
		T:               ev.Time.Sub(begin).Seconds(),
	}
}

// NewNetworkEventsList returns a list of network events.
func NewNetworkEventsList(begin time.Time, events []Event) (out []NetworkEvent) {
	for _, wrapper := range events {
		ev := wrapper.Value()
		switch wrapper.(type) {
		case *EventConnectOperation:
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				Proto:     ev.Proto,
				T:         ev.Time.Sub(begin).Seconds(),
			})
		case *EventReadOperation:
			out = append(out, NetworkEvent{
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
		case *EventWriteOperation:
			out = append(out, NetworkEvent{
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
		case *EventReadFromOperation:
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
		case *EventWriteToOperation:
			out = append(out, NetworkEvent{
				Address:   ev.Address,
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				NumBytes:  int64(ev.NumBytes),
				T:         ev.Time.Sub(begin).Seconds(),
			})
		default: // For example, "tls_handshake_done" (used in data analysis!)
			out = append(out, NetworkEvent{
				Failure:   ev.Err.ToFailure(),
				Operation: wrapper.Name(),
				T:         ev.Time.Sub(begin).Seconds(),
			})
		}
	}
	return
}

// NewTLSHandshakesList creates a new TLSHandshakesList
func NewTLSHandshakesList(begin time.Time, events []Event) (out []TLSHandshake) {
	for _, wrapper := range events {
		switch wrapper.(type) {
		case *EventQUICHandshakeDone, *EventTLSHandshakeDone: // interested
		default:
			continue // not interested
		}
		ev := wrapper.Value()
		out = append(out, TLSHandshake{
			Address:            ev.Address,
			CipherSuite:        ev.TLSCipherSuite,
			Failure:            ev.Err.ToFailure(),
			NegotiatedProtocol: ev.TLSNegotiatedProto,
			NoTLSVerify:        ev.NoTLSVerify,
			PeerCertificates:   tlsMakePeerCerts(ev.TLSPeerCerts),
			ServerName:         ev.TLSServerName,
			T:                  ev.Time.Sub(begin).Seconds(),
			TLSVersion:         ev.TLSVersion,
		})
	}
	return
}

func tlsMakePeerCerts(in [][]byte) (out []MaybeBinaryValue) {
	for _, entry := range in {
		out = append(out, MaybeBinaryValue{Value: string(entry)})
	}
	return
}
