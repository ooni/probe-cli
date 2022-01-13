package archival

import (
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

//
// Trace implementation
//

// Trace contains the events.
type Trace struct {
	// DNSLookupHTTPS contains DNSLookupHTTPS events.
	DNSLookupHTTPS []*DNSLookupEvent

	// DNSLookupHost contains DNSLookupHost events.
	DNSLookupHost []*DNSLookupEvent

	// DNSRoundTrip contains DNSRoundTrip events.
	DNSRoundTrip []*DNSRoundTripEvent

	// HTTPRoundTrip contains HTTPRoundTrip round trip events.
	HTTPRoundTrip []*HTTPRoundTripEvent

	// Network contains network events.
	Network []*NetworkEvent

	// QUICHandshake contains QUICHandshake handshake events.
	QUICHandshake []*QUICTLSHandshakeEvent

	// TLSHandshake contains TLSHandshake handshake events.
	TLSHandshake []*QUICTLSHandshakeEvent
}

func (t *Trace) newFailure(err error) (out *string) {
	if err != nil {
		s := err.Error()
		out = &s
	}
	return
}

//
// TCP connect
//

// NewArchivalTCPConnectResultList builds a TCP connect list in the OONI archival
// data format out of the results saved inside the trace.
func (t *Trace) NewArchivalTCPConnectResultList(begin time.Time) (out []model.ArchivalTCPConnectResult) {
	for _, ev := range t.Network {
		if ev.Operation != netxlite.ConnectOperation || ev.Network != "tcp" {
			continue
		}
		// We assume Go is passing us legit data structures
		ip, sport, _ := net.SplitHostPort(ev.RemoteAddr)
		iport, _ := strconv.Atoi(sport)
		out = append(out, model.ArchivalTCPConnectResult{
			IP:   ip,
			Port: iport,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: nil, // Web Connectivity only, depends on the control
				Failure: t.newFailure(ev.Failure),
				Success: ev.Failure == nil,
			},
			T: ev.Finished.Sub(begin).Seconds(),
		})
	}
	return
}

//
// HTTP
//

// NewArchivalHTTPRequestResultList builds an HTTP requests list in the OONI
// archival data format out of the results saved inside the trace.
//
// This function will sort the emitted list of requests such that the last
// request that happened in time is the first one to be emitted. If the
// measurement code performs related requests sequentially (which is a kinda a
// given because you cannot follow a redirect before reading the previous request),
// then the result is sorted how the OONI pipeline expects it to be.
func (t *Trace) NewArchivalHTTPRequestResultList(begin time.Time) (out []model.ArchivalHTTPRequestResult) {
	for _, ev := range t.HTTPRoundTrip {
		out = append(out, model.ArchivalHTTPRequestResult{
			Failure: t.newFailure(ev.Failure),
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList:     t.newHTTPHeadersList(ev.RequestHeaders),
				Headers:         t.newHTTPHeadersMap(ev.RequestHeaders),
				Method:          ev.Method,
				Tor:             model.ArchivalHTTPTor{},
				Transport:       ev.Transport,
				URL:             ev.URL,
			},
			Response: model.ArchivalHTTPResponse{
				Body: model.ArchivalMaybeBinaryData{
					Value: string(ev.ResponseBody),
				},
				BodyIsTruncated: ev.ResponseBodyIsTruncated,
				Code:            ev.StatusCode,
				HeadersList:     t.newHTTPHeadersList(ev.ResponseHeaders),
				Headers:         t.newHTTPHeadersMap(ev.ResponseHeaders),
				Locations:       ev.ResponseHeaders.Values("Location"), // safe with nil headers
			},
			T: ev.Finished.Sub(begin).Seconds(),
		})
	}
	// Implementation note: historically OONI has always added
	// the _last_ measurement in _first_ position. This has only
	// been relevant for sequentially performed requests. For
	// this purpose it feels okay to use T as the sorting key,
	// since it's the time when we exited RoundTrip().
	sort.Slice(out, func(i, j int) bool {
		return out[i].T > out[j].T
	})
	return
}

func (t *Trace) newHTTPHeadersList(source http.Header) (out []model.ArchivalHTTPHeader) {
	for key, values := range source {
		for _, value := range values {
			out = append(out, model.ArchivalHTTPHeader{
				Key: key,
				Value: model.ArchivalMaybeBinaryData{
					Value: value,
				},
			})
		}
	}
	// Implementation note: we need to sort the keys to have
	// stable testing since map iteration is random.
	sort.Slice(out, func(i, j int) bool {
		return out[i].Key < out[j].Key
	})
	return
}

func (t *Trace) newHTTPHeadersMap(source http.Header) (out map[string]model.ArchivalMaybeBinaryData) {
	for key, values := range source {
		for index, value := range values {
			if index > 0 {
				break // only the first entry
			}
			if out == nil {
				out = make(map[string]model.ArchivalMaybeBinaryData)
			}
			out[key] = model.ArchivalMaybeBinaryData{Value: value}
		}
	}
	return
}

//
// DNS
//

// NewArchivalDNSLookupResultList builds a DNS lookups list in the OONI
// archival data format out of the results saved inside the trace.
func (t *Trace) NewArchivalDNSLookupResultList(begin time.Time) (out []model.ArchivalDNSLookupResult) {
	for _, ev := range t.DNSLookupHost {
		out = append(out, model.ArchivalDNSLookupResult{
			Answers:          t.gatherA(ev.Addresses),
			Engine:           ev.ResolverNetwork,
			Failure:          t.newFailure(ev.Failure),
			Hostname:         ev.Domain,
			QueryType:        "A",
			ResolverHostname: nil, // legacy
			ResolverPort:     nil, // legacy
			ResolverAddress:  ev.ResolverAddress,
			T:                ev.Finished.Sub(begin).Seconds(),
		})
		aaaa := t.gatherAAAA(ev.Addresses)
		if len(aaaa) <= 0 && ev.Failure == nil {
			// We don't have any AAAA results. Historically we do not
			// create a record for AAAA with no results when A succeeded
			continue
		}
		out = append(out, model.ArchivalDNSLookupResult{
			Answers:          aaaa,
			Engine:           ev.ResolverNetwork,
			Failure:          t.newFailure(ev.Failure),
			Hostname:         ev.Domain,
			QueryType:        "AAAA",
			ResolverHostname: nil, // legacy
			ResolverPort:     nil, // legacy
			ResolverAddress:  ev.ResolverAddress,
			T:                ev.Finished.Sub(begin).Seconds(),
		})
	}
	return
}

func (t *Trace) gatherA(addrs []string) (out []model.ArchivalDNSAnswer) {
	for _, addr := range addrs {
		if strings.Contains(addr, ":") {
			continue // it's AAAA so we need to skip it
		}
		answer := model.ArchivalDNSAnswer{AnswerType: "A"}
		asn, org, _ := geolocate.LookupASN(addr)
		answer.ASN = int64(asn)
		answer.ASOrgName = org
		answer.IPv4 = addr
		out = append(out, answer)
	}
	return
}

func (t *Trace) gatherAAAA(addrs []string) (out []model.ArchivalDNSAnswer) {
	for _, addr := range addrs {
		if !strings.Contains(addr, ":") {
			continue // it's A so we need to skip it
		}
		answer := model.ArchivalDNSAnswer{AnswerType: "AAAA"}
		asn, org, _ := geolocate.LookupASN(addr)
		answer.ASN = int64(asn)
		answer.ASOrgName = org
		answer.IPv6 = addr
		out = append(out, answer)
	}
	return
}

//
// NetworkEvents
//

// NewArchivalNetworkEventList builds a network events list in the OONI
// archival data format out of the results saved inside the trace.
func (t *Trace) NewArchivalNetworkEventList(begin time.Time) (out []model.ArchivalNetworkEvent) {
	for _, ev := range t.Network {
		out = append(out, model.ArchivalNetworkEvent{
			Address:   ev.RemoteAddr,
			Failure:   t.newFailure(ev.Failure),
			NumBytes:  int64(ev.Count),
			Operation: ev.Operation,
			Proto:     ev.Network,
			T:         ev.Finished.Sub(begin).Seconds(),
			Tags:      nil,
		})
	}
	return
}

//
// TLS handshake
//

// NewArchivalTLSHandshakeResultList builds a TLS handshakes list in the OONI
// archival data format out of the results saved inside the trace.
func (t *Trace) NewArchivalTLSHandshakeResultList(begin time.Time) (out []model.ArchivalTLSOrQUICHandshakeResult) {
	for _, ev := range t.TLSHandshake {
		out = append(out, model.ArchivalTLSOrQUICHandshakeResult{
			CipherSuite:        ev.CipherSuite,
			Failure:            t.newFailure(ev.Failure),
			NegotiatedProtocol: ev.NegotiatedProto,
			NoTLSVerify:        ev.SkipVerify,
			PeerCertificates:   t.makePeerCerts(ev.PeerCerts),
			ServerName:         ev.SNI,
			T:                  ev.Finished.Sub(begin).Seconds(),
			Tags:               nil,
			TLSVersion:         ev.TLSVersion,
		})
	}
	return
}

func (t *Trace) makePeerCerts(in [][]byte) (out []model.ArchivalMaybeBinaryData) {
	for _, v := range in {
		out = append(out, model.ArchivalMaybeBinaryData{Value: string(v)})
	}
	return
}
