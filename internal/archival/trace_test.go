package archival

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// traceTime generates a time that is based off a fixed beginning
// in time, so we can easily compare times.
func traceTime(d int64) time.Time {
	t := time.Date(2021, 01, 13, 14, 21, 59, 0, time.UTC)
	return t.Add(time.Duration(d) * time.Millisecond)
}

// deltaSinceTraceTime computes the delta since the original
// trace time expressed in floating point seconds.
func deltaSinceTraceTime(d int64) float64 {
	return (time.Duration(d) * time.Millisecond).Seconds()
}

// failureFromString converts a string to a failure.
func failureFromString(failure string) *string {
	return &failure
}

func TestTraceNewArchivalTCPConnectResultList(t *testing.T) {
	type fields struct {
		DNSLookupHTTPS []*DNSLookupEvent
		DNSLookupHost  []*DNSLookupEvent
		DNSRoundTrip   []*DNSRoundTripEvent
		HTTPRoundTrip  []*HTTPRoundTripEvent
		Network        []*NetworkEvent
		QUICHandshake  []*QUICTLSHandshakeEvent
		TLSHandshake   []*QUICTLSHandshakeEvent
	}
	type args struct {
		begin time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOut []model.ArchivalTCPConnectResult
	}{{
		name: "with empty trace",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network:        []*NetworkEvent{},
			QUICHandshake:  []*QUICTLSHandshakeEvent{},
			TLSHandshake:   []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: nil,
	}, {
		name: "we ignore I/O operations",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network: []*NetworkEvent{{
				Count:      1024,
				Failure:    nil,
				Finished:   traceTime(2),
				Network:    "tcp",
				Operation:  netxlite.WriteOperation,
				RemoteAddr: "8.8.8.8:443",
				Started:    traceTime(1),
			}, {
				Count:      4096,
				Failure:    nil,
				Finished:   traceTime(4),
				Network:    "tcp",
				Operation:  netxlite.ReadOperation,
				RemoteAddr: "8.8.8.8:443",
				Started:    traceTime(3),
			}},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: nil,
	}, {
		name: "we ignore UDP connect",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network: []*NetworkEvent{{
				Count:      0,
				Failure:    nil,
				Finished:   traceTime(2),
				Network:    "udp",
				Operation:  netxlite.ConnectOperation,
				RemoteAddr: "8.8.8.8:53",
				Started:    traceTime(1),
			}},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: nil,
	}, {
		name: "with TCP connect success",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network: []*NetworkEvent{{
				Count:      0,
				Failure:    nil,
				Finished:   traceTime(2),
				Network:    "tcp",
				Operation:  netxlite.ConnectOperation,
				RemoteAddr: "8.8.8.8:443",
				Started:    traceTime(1),
			}},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalTCPConnectResult{{
			IP:   "8.8.8.8",
			Port: 443,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: nil,
				Failure: nil,
				Success: true,
			},
			T: deltaSinceTraceTime(2),
		}},
	}, {
		name: "with TCP connect failure",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network: []*NetworkEvent{{
				Count:      0,
				Failure:    netxlite.NewTopLevelGenericErrWrapper(netxlite.ECONNREFUSED),
				Finished:   traceTime(2),
				Network:    "tcp",
				Operation:  netxlite.ConnectOperation,
				RemoteAddr: "8.8.8.8:443",
				Started:    traceTime(1),
			}},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalTCPConnectResult{{
			IP:   "8.8.8.8",
			Port: 443,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: nil,
				Failure: failureFromString(netxlite.FailureConnectionRefused),
				Success: false,
			},
			T: deltaSinceTraceTime(2),
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trace{
				DNSLookupHTTPS: tt.fields.DNSLookupHTTPS,
				DNSLookupHost:  tt.fields.DNSLookupHost,
				DNSRoundTrip:   tt.fields.DNSRoundTrip,
				HTTPRoundTrip:  tt.fields.HTTPRoundTrip,
				Network:        tt.fields.Network,
				QUICHandshake:  tt.fields.QUICHandshake,
				TLSHandshake:   tt.fields.TLSHandshake,
			}
			gotOut := tr.NewArchivalTCPConnectResultList(tt.args.begin)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTraceNewArchivalHTTPRequestResultList(t *testing.T) {
	type fields struct {
		DNSLookupHTTPS []*DNSLookupEvent
		DNSLookupHost  []*DNSLookupEvent
		DNSRoundTrip   []*DNSRoundTripEvent
		HTTPRoundTrip  []*HTTPRoundTripEvent
		Network        []*NetworkEvent
		QUICHandshake  []*QUICTLSHandshakeEvent
		TLSHandshake   []*QUICTLSHandshakeEvent
	}
	type args struct {
		begin time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOut []model.ArchivalHTTPRequestResult
	}{{
		name: "with empty trace",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network:        []*NetworkEvent{},
			QUICHandshake:  []*QUICTLSHandshakeEvent{},
			TLSHandshake:   []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: nil,
	}, {
		name: "with failure",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{{
				Failure:  netxlite.NewTopLevelGenericErrWrapper(netxlite.ECONNRESET),
				Finished: traceTime(2),
				Method:   "GET",
				RequestHeaders: http.Header{
					"Accept":   {"*/*"},
					"X-Cookie": {"A", "B", "C"},
				},
				ResponseBody:            nil,
				ResponseBodyIsTruncated: false,
				ResponseBodyLength:      0,
				ResponseHeaders:         nil,
				Started:                 traceTime(1),
				StatusCode:              0,
				Transport:               "tcp",
				URL:                     "http://x.org/",
			}},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalHTTPRequestResult{{
			Failure: failureFromString(netxlite.FailureConnectionReset),
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Accept",
					Value: model.ArchivalMaybeBinaryData{
						Value: "*/*",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "A",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "B",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "C",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Accept":   {Value: "*/*"},
					"X-Cookie": {Value: "A"},
				},
				Method:    "GET",
				Tor:       model.ArchivalHTTPTor{},
				Transport: "tcp",
				URL:       "http://x.org/",
			},
			Response: model.ArchivalHTTPResponse{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				Code:            0,
				HeadersList:     nil,
				Headers:         nil,
				Locations:       nil,
			},
			T: deltaSinceTraceTime(2),
		}},
	}, {
		name: "with success",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{{
				Failure:  nil,
				Finished: traceTime(2),
				Method:   "GET",
				RequestHeaders: http.Header{
					"Accept":   {"*/*"},
					"X-Cookie": {"A", "B", "C"},
				},
				ResponseBody:            []byte("0xdeadbeef"),
				ResponseBodyIsTruncated: true,
				ResponseBodyLength:      10,
				ResponseHeaders: http.Header{
					"Server":         {"antani/1.0"},
					"X-Cookie-Reply": {"C", "D", "F"},
					"Location":       {"https://x.org/", "https://x.org/robots.txt"},
				},
				Started:    traceTime(1),
				StatusCode: 302,
				Transport:  "tcp",
				URL:        "http://x.org/",
			}},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalHTTPRequestResult{{
			Failure: nil,
			Request: model.ArchivalHTTPRequest{
				Body:            model.ArchivalMaybeBinaryData{},
				BodyIsTruncated: false,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Accept",
					Value: model.ArchivalMaybeBinaryData{
						Value: "*/*",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "A",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "B",
					},
				}, {
					Key: "X-Cookie",
					Value: model.ArchivalMaybeBinaryData{
						Value: "C",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Accept":   {Value: "*/*"},
					"X-Cookie": {Value: "A"},
				},
				Method:    "GET",
				Tor:       model.ArchivalHTTPTor{},
				Transport: "tcp",
				URL:       "http://x.org/",
			},
			Response: model.ArchivalHTTPResponse{
				Body: model.ArchivalMaybeBinaryData{
					Value: "0xdeadbeef",
				},
				BodyIsTruncated: true,
				Code:            302,
				HeadersList: []model.ArchivalHTTPHeader{{
					Key: "Location",
					Value: model.ArchivalMaybeBinaryData{
						Value: "https://x.org/",
					},
				}, {
					Key: "Location",
					Value: model.ArchivalMaybeBinaryData{
						Value: "https://x.org/robots.txt",
					},
				}, {
					Key: "Server",
					Value: model.ArchivalMaybeBinaryData{
						Value: "antani/1.0",
					},
				}, {
					Key: "X-Cookie-Reply",
					Value: model.ArchivalMaybeBinaryData{
						Value: "C",
					},
				}, {
					Key: "X-Cookie-Reply",
					Value: model.ArchivalMaybeBinaryData{
						Value: "D",
					},
				}, {
					Key: "X-Cookie-Reply",
					Value: model.ArchivalMaybeBinaryData{
						Value: "F",
					},
				}},
				Headers: map[string]model.ArchivalMaybeBinaryData{
					"Server":         {Value: "antani/1.0"},
					"X-Cookie-Reply": {Value: "C"},
					"Location":       {Value: "https://x.org/"},
				},
				Locations: []string{
					"https://x.org/",
					"https://x.org/robots.txt",
				},
			},
			T: deltaSinceTraceTime(2),
		}},
	}, {
		name: "The result is sorted by the value of T",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{{
				Failure:                 nil,
				Finished:                traceTime(3),
				Method:                  "",
				RequestHeaders:          map[string][]string{},
				ResponseBody:            []byte{},
				ResponseBodyIsTruncated: false,
				ResponseBodyLength:      0,
				ResponseHeaders:         map[string][]string{},
				Started:                 time.Time{},
				StatusCode:              0,
				Transport:               "",
				URL:                     "",
			}, {
				Failure:                 nil,
				Finished:                traceTime(2),
				Method:                  "",
				RequestHeaders:          map[string][]string{},
				ResponseBody:            []byte{},
				ResponseBodyIsTruncated: false,
				ResponseBodyLength:      0,
				ResponseHeaders:         map[string][]string{},
				Started:                 time.Time{},
				StatusCode:              0,
				Transport:               "",
				URL:                     "",
			}, {
				Failure:                 nil,
				Finished:                traceTime(5),
				Method:                  "",
				RequestHeaders:          map[string][]string{},
				ResponseBody:            []byte{},
				ResponseBodyIsTruncated: false,
				ResponseBodyLength:      0,
				ResponseHeaders:         map[string][]string{},
				Started:                 time.Time{},
				StatusCode:              0,
				Transport:               "",
				URL:                     "",
			}},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalHTTPRequestResult{{
			Failure:  nil,
			Request:  model.ArchivalHTTPRequest{},
			Response: model.ArchivalHTTPResponse{},
			T:        deltaSinceTraceTime(5),
		}, {
			Failure:  nil,
			Request:  model.ArchivalHTTPRequest{},
			Response: model.ArchivalHTTPResponse{},
			T:        deltaSinceTraceTime(3),
		}, {
			Failure:  nil,
			Request:  model.ArchivalHTTPRequest{},
			Response: model.ArchivalHTTPResponse{},
			T:        deltaSinceTraceTime(2),
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trace{
				DNSLookupHTTPS: tt.fields.DNSLookupHTTPS,
				DNSLookupHost:  tt.fields.DNSLookupHost,
				DNSRoundTrip:   tt.fields.DNSRoundTrip,
				HTTPRoundTrip:  tt.fields.HTTPRoundTrip,
				Network:        tt.fields.Network,
				QUICHandshake:  tt.fields.QUICHandshake,
				TLSHandshake:   tt.fields.TLSHandshake,
			}
			gotOut := tr.NewArchivalHTTPRequestResultList(tt.args.begin)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTraceNewArchivalDNSLookupResultList(t *testing.T) {
	type fields struct {
		DNSLookupHTTPS []*DNSLookupEvent
		DNSLookupHost  []*DNSLookupEvent
		DNSRoundTrip   []*DNSRoundTripEvent
		HTTPRoundTrip  []*HTTPRoundTripEvent
		Network        []*NetworkEvent
		QUICHandshake  []*QUICTLSHandshakeEvent
		TLSHandshake   []*QUICTLSHandshakeEvent
	}
	type args struct {
		begin time.Time
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantOut []model.ArchivalDNSLookupResult
	}{{
		name: "with empty trace",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost:  []*DNSLookupEvent{},
			DNSRoundTrip:   []*DNSRoundTripEvent{},
			HTTPRoundTrip:  []*HTTPRoundTripEvent{},
			Network:        []*NetworkEvent{},
			QUICHandshake:  []*QUICTLSHandshakeEvent{},
			TLSHandshake:   []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: nil,
	}, {
		name: "with NXDOMAIN failure",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost: []*DNSLookupEvent{{
				ALPNs:           nil,
				Addresses:       nil,
				Domain:          "example.com",
				Failure:         netxlite.NewTopLevelGenericErrWrapper(errors.New(netxlite.DNSNoSuchHostSuffix)),
				Finished:        traceTime(2),
				LookupType:      "", // not processed
				ResolverAddress: "8.8.8.8:53",
				ResolverNetwork: "udp",
				Started:         traceTime(1),
			}},
			DNSRoundTrip:  []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalDNSLookupResult{{
			Answers:          nil,
			Engine:           "udp",
			Failure:          failureFromString(netxlite.FailureDNSNXDOMAINError),
			Hostname:         "example.com",
			QueryType:        "A",
			ResolverHostname: nil,
			ResolverPort:     nil,
			ResolverAddress:  "8.8.8.8:53",
			T:                deltaSinceTraceTime(2),
		}, {
			Answers:          nil,
			Engine:           "udp",
			Failure:          failureFromString(netxlite.FailureDNSNXDOMAINError),
			Hostname:         "example.com",
			QueryType:        "AAAA",
			ResolverHostname: nil,
			ResolverPort:     nil,
			ResolverAddress:  "8.8.8.8:53",
			T:                deltaSinceTraceTime(2),
		}},
	}, {
		name: "with success for A and AAAA",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost: []*DNSLookupEvent{{
				ALPNs: nil,
				Addresses: []string{
					"8.8.8.8", "8.8.4.4", "2001:4860:4860::8844",
				},
				Domain:          "dns.google",
				Failure:         nil,
				Finished:        traceTime(2),
				LookupType:      "", // not processed
				ResolverAddress: "8.8.8.8:53",
				ResolverNetwork: "udp",
				Started:         traceTime(1),
			}},
			DNSRoundTrip:  []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalDNSLookupResult{{
			Answers: []model.ArchivalDNSAnswer{{
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				Hostname:   "",
				IPv4:       "8.8.8.8",
				IPv6:       "",
				TTL:        nil,
			}, {
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				Hostname:   "",
				IPv4:       "8.8.4.4",
				IPv6:       "",
				TTL:        nil,
			}},
			Engine:           "udp",
			Failure:          nil,
			Hostname:         "dns.google",
			QueryType:        "A",
			ResolverHostname: nil,
			ResolverPort:     nil,
			ResolverAddress:  "8.8.8.8:53",
			T:                deltaSinceTraceTime(2),
		}, {
			Answers: []model.ArchivalDNSAnswer{{
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "AAAA",
				Hostname:   "",
				IPv4:       "",
				IPv6:       "2001:4860:4860::8844",
				TTL:        nil,
			}},
			Engine:           "udp",
			Failure:          nil,
			Hostname:         "dns.google",
			QueryType:        "AAAA",
			ResolverHostname: nil,
			ResolverPort:     nil,
			ResolverAddress:  "8.8.8.8:53",
			T:                deltaSinceTraceTime(2),
		}},
	}, {
		name: "when a domain has no AAAA addresses",
		fields: fields{
			DNSLookupHTTPS: []*DNSLookupEvent{},
			DNSLookupHost: []*DNSLookupEvent{{
				ALPNs: nil,
				Addresses: []string{
					"8.8.8.8", "8.8.4.4",
				},
				Domain:          "dns.google",
				Failure:         nil,
				Finished:        traceTime(2),
				LookupType:      "", // not processed
				ResolverAddress: "8.8.8.8:53",
				ResolverNetwork: "udp",
				Started:         traceTime(1),
			}},
			DNSRoundTrip:  []*DNSRoundTripEvent{},
			HTTPRoundTrip: []*HTTPRoundTripEvent{},
			Network:       []*NetworkEvent{},
			QUICHandshake: []*QUICTLSHandshakeEvent{},
			TLSHandshake:  []*QUICTLSHandshakeEvent{},
		},
		args: args{
			begin: traceTime(0),
		},
		wantOut: []model.ArchivalDNSLookupResult{{
			Answers: []model.ArchivalDNSAnswer{{
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				Hostname:   "",
				IPv4:       "8.8.8.8",
				IPv6:       "",
				TTL:        nil,
			}, {
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				Hostname:   "",
				IPv4:       "8.8.4.4",
				IPv6:       "",
				TTL:        nil,
			}},
			Engine:           "udp",
			Failure:          nil,
			Hostname:         "dns.google",
			QueryType:        "A",
			ResolverHostname: nil,
			ResolverPort:     nil,
			ResolverAddress:  "8.8.8.8:53",
			T:                deltaSinceTraceTime(2),
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trace{
				DNSLookupHTTPS: tt.fields.DNSLookupHTTPS,
				DNSLookupHost:  tt.fields.DNSLookupHost,
				DNSRoundTrip:   tt.fields.DNSRoundTrip,
				HTTPRoundTrip:  tt.fields.HTTPRoundTrip,
				Network:        tt.fields.Network,
				QUICHandshake:  tt.fields.QUICHandshake,
				TLSHandshake:   tt.fields.TLSHandshake,
			}
			gotOut := tr.NewArchivalDNSLookupResultList(tt.args.begin)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTraceNewArchivalNetworkEventList(t *testing.T) {
}

func TestTraceNewArchivalNetworkEventListWithTags(t *testing.T) {
}

func TestTraceNewArchivalTLSHandshakeResultList(t *testing.T) {
}

func TestTraceNewArchivalTLSHandshakeResultListWithTags(t *testing.T) {
}
