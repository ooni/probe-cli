package tracex

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSQueryType(t *testing.T) {
	t.Run("ipOfType", func(t *testing.T) {
		type expectation struct {
			qtype  dnsQueryType
			ip     string
			output bool
		}
		var expectations = []expectation{{
			qtype:  "A",
			ip:     "8.8.8.8",
			output: true,
		}, {
			qtype:  "A",
			ip:     "2a00:1450:4002:801::2004",
			output: false,
		}, {
			qtype:  "AAAA",
			ip:     "8.8.8.8",
			output: false,
		}, {
			qtype:  "AAAA",
			ip:     "2a00:1450:4002:801::2004",
			output: true,
		}, {
			qtype:  "ANTANI",
			ip:     "2a00:1450:4002:801::2004",
			output: false,
		}, {
			qtype:  "ANTANI",
			ip:     "8.8.8.8",
			output: false,
		}}
		for _, exp := range expectations {
			if exp.qtype.ipOfType(exp.ip) != exp.output {
				t.Fatalf("failure for %+v", exp)
			}
		}
	})
}

func TestNewTCPConnectList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []Event
	}
	tests := []struct {
		name string
		args args
		want []TCPConnectEntry
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin: begin,
			events: []Event{&EventResolveDone{&EventValue{ // skipped because not relevant
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "dns.google.com",
				Time:      begin.Add(100 * time.Millisecond),
			}}, &EventConnectOperation{&EventValue{
				Address:  "8.8.8.8:853",
				Duration: 30 * time.Millisecond,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}}, &EventConnectOperation{&EventValue{
				Address:  "8.8.8.8:853",
				Duration: 55 * time.Millisecond,
				Proto:    "udp", // this one should be skipped because it's UDP
				Time:     begin.Add(130 * time.Millisecond),
			}}, &EventConnectOperation{&EventValue{
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Proto:    "tcp",
				Time:     begin.Add(180 * time.Millisecond),
			}}},
		},
		want: []TCPConnectEntry{{
			IP:   "8.8.8.8",
			Port: 853,
			Status: TCPConnectStatus{
				Success: true,
			},
			T: 0.13,
		}, {
			IP:   "8.8.4.4",
			Port: 53,
			Status: TCPConnectStatus{
				Failure: NewFailure(io.EOF),
				Success: false,
			},
			T: 0.18,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTCPConnectList(tt.args.begin, tt.args.events)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewRequestList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []Event
	}
	tests := []struct {
		name string
		args args
		want []RequestEntry
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin: begin,
			// Two round trips so we can test the sorting expected by OONI
			events: []Event{&EventHTTPTransactionDone{&EventValue{
				HTTPRequestHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "POST",
				HTTPURL:    "https://www.example.com/submit",
				HTTPResponseHeaders: http.Header{
					"Server": []string{"miniooni/0.1.0-dev"},
				},
				HTTPStatusCode:              200,
				HTTPResponseBody:            []byte("{}"),
				HTTPResponseBodyIsTruncated: false,
				Time:                        begin.Add(10 * time.Millisecond),
			}}, &EventHTTPTransactionDone{&EventValue{
				HTTPRequestHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "GET",
				HTTPURL:    "https://www.example.com/result",
				Err:        io.EOF,
				Time:       begin.Add(20 * time.Millisecond),
			}}},
		},
		want: []RequestEntry{{
			Failure: NewFailure(io.EOF),
			Request: HTTPRequest{
				HeadersList: []HTTPHeader{{
					Key: "User-Agent",
					Value: MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "GET",
				URL:    "https://www.example.com/result",
			},
			Response: HTTPResponse{
				HeadersList: []HTTPHeader{},
				Headers:     make(map[string]MaybeBinaryValue),
			},
			T: 0.02,
		}, {
			Request: HTTPRequest{
				Body: MaybeBinaryValue{
					Value: "",
				},
				HeadersList: []HTTPHeader{{
					Key: "User-Agent",
					Value: MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "POST",
				URL:    "https://www.example.com/submit",
			},
			Response: HTTPResponse{
				Body: MaybeBinaryValue{
					Value: "{}",
				},
				Code: 200,
				HeadersList: []HTTPHeader{{
					Key: "Server",
					Value: MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]MaybeBinaryValue{
					"Server": {Value: "miniooni/0.1.0-dev"},
				},
				Locations: nil,
			},
			T: 0.01,
		}},
	}, {
		// for an example of why we need to sort headers, see
		// https://github.com/ooni/probe-engine/pull/751/checks?check_run_id=853562310
		name: "run with redirect and headers to sort",
		args: args{
			begin: begin,
			events: []Event{&EventHTTPTransactionDone{&EventValue{
				HTTPRequestHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "GET",
				HTTPURL:    "https://www.example.com/",
				HTTPResponseHeaders: http.Header{
					"Server":   []string{"miniooni/0.1.0-dev"},
					"Location": []string{"https://x.example.com", "https://y.example.com"},
				},
				HTTPStatusCode: 302,
				Time:           begin.Add(10 * time.Millisecond),
			}}},
		},
		want: []RequestEntry{{
			Request: HTTPRequest{
				HeadersList: []HTTPHeader{{
					Key: "User-Agent",
					Value: MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "GET",
				URL:    "https://www.example.com/",
			},
			Response: HTTPResponse{
				Code: 302,
				HeadersList: []HTTPHeader{{
					Key: "Location",
					Value: MaybeBinaryValue{
						Value: "https://x.example.com",
					},
				}, {
					Key: "Location",
					Value: MaybeBinaryValue{
						Value: "https://y.example.com",
					},
				}, {
					Key: "Server",
					Value: MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]MaybeBinaryValue{
					"Server":   {Value: "miniooni/0.1.0-dev"},
					"Location": {Value: "https://x.example.com"},
				},
				Locations: []string{
					"https://x.example.com", "https://y.example.com",
				},
			},
			T: 0.01,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRequestList(tt.args.begin, tt.args.events)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewDNSQueriesList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []Event
	}
	tests := []struct {
		name string
		args args
		want []DNSQueryEntry
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin: begin,
			events: []Event{&EventResolveDone{&EventValue{
				Address:   "1.1.1.1:853",
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "dns.google.com",
				Proto:     "dot",
				Time:      begin.Add(100 * time.Millisecond),
			}}, &EventConnectOperation{&EventValue{ // skipped because not relevant
				Address:  "8.8.8.8:853",
				Duration: 30 * time.Millisecond,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}}, &EventConnectOperation{&EventValue{ // skipped because not relevant
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Proto:    "tcp",
				Time:     begin.Add(180 * time.Millisecond),
			}}},
		},
		want: []DNSQueryEntry{{
			Answers: []DNSAnswerEntry{{
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				IPv4:       "8.8.8.8",
			}, {
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "A",
				IPv4:       "8.8.4.4",
			}},
			Engine:          "dot",
			Hostname:        "dns.google.com",
			QueryType:       "A",
			ResolverAddress: "1.1.1.1:853",
			T:               0.1,
		}},
	}, {
		name: "run with IPv6 results",
		args: args{
			begin: begin,
			events: []Event{&EventResolveDone{&EventValue{
				Addresses: []string{"2001:4860:4860::8888"},
				Hostname:  "dns.google.com",
				Time:      begin.Add(200 * time.Millisecond),
			}}},
		},
		want: []DNSQueryEntry{{
			Answers: []DNSAnswerEntry{{
				ASN:        15169,
				ASOrgName:  "Google LLC",
				AnswerType: "AAAA",
				IPv6:       "2001:4860:4860::8888",
			}},
			Hostname:  "dns.google.com",
			QueryType: "AAAA",
			T:         0.2,
		}},
	}, {
		name: "run with errors",
		args: args{
			begin: begin,
			events: []Event{&EventResolveDone{&EventValue{
				Err:      &netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError},
				Hostname: "dns.google.com",
				Time:     begin.Add(200 * time.Millisecond),
			}}},
		},
		want: []DNSQueryEntry{{
			Answers: nil,
			Failure: NewFailure(
				&netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError}),
			Hostname:  "dns.google.com",
			QueryType: "A",
			T:         0.2,
		}, {
			Answers: nil,
			Failure: NewFailure(
				&netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError}),
			Hostname:  "dns.google.com",
			QueryType: "AAAA",
			T:         0.2,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewDNSQueriesList(tt.args.begin, tt.args.events)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewNetworkEventsList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []Event
	}
	tests := []struct {
		name string
		args args
		want []NetworkEvent
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin: begin,
			events: []Event{&EventConnectOperation{&EventValue{
				Address: "8.8.8.8:853",
				Err:     io.EOF,
				Proto:   "tcp",
				Time:    begin.Add(7 * time.Millisecond),
			}}, &EventReadOperation{&EventValue{
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}}, &EventReadFromOperation{&EventValue{
				Address:  "8.8.8.8:853",
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}}, &EventWriteOperation{&EventValue{
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}}, &EventWriteToOperation{&EventValue{
				Address:  "8.8.8.8:853",
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}}, &EventResolveStart{&EventValue{
				// We expect this event to be logged event though it's not a typical I/O
				// event (it seems these extra events are useful for debugging)
				Time: begin.Add(15 * time.Millisecond),
			}}},
		},
		want: []NetworkEvent{{
			Address:   "8.8.8.8:853",
			Failure:   NewFailure(io.EOF),
			Operation: netxlite.ConnectOperation,
			Proto:     "tcp",
			T:         0.007,
		}, {
			Failure:   NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: netxlite.ReadOperation,
			T:         0.011,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: netxlite.ReadFromOperation,
			T:         0.011,
		}, {
			Failure:   NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: netxlite.WriteOperation,
			T:         0.014,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: netxlite.WriteToOperation,
			T:         0.014,
		}, {
			Operation: "resolve_start",
			T:         0.015,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewNetworkEventsList(tt.args.begin, tt.args.events)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewTLSHandshakesList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []Event
	}
	tests := []struct {
		name string
		args args
		want []TLSHandshake
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run with TLS",
		args: args{
			begin: begin,
			events: []Event{&EventTLSHandshakeDone{&EventValue{
				Address:            "131.252.210.176:443",
				Err:                io.EOF,
				NoTLSVerify:        false,
				Proto:              "tcp",
				TLSCipherSuite:     "SUITE",
				TLSNegotiatedProto: "h2",
				TLSPeerCerts: []*x509.Certificate{{
					Raw: []byte("deadbeef"),
				}, {
					Raw: []byte("abad1dea"),
				}},
				TLSServerName: "x.org",
				TLSVersion:    "TLSv1.3",
				Time:          begin.Add(55 * time.Millisecond),
			}}},
		},
		want: []TLSHandshake{{
			Address:            "131.252.210.176:443",
			CipherSuite:        "SUITE",
			Failure:            NewFailure(io.EOF),
			NegotiatedProtocol: "h2",
			NoTLSVerify:        false,
			PeerCertificates: []MaybeBinaryValue{{
				Value: "deadbeef",
			}, {
				Value: "abad1dea",
			}},
			ServerName: "x.org",
			T:          0.055,
			TLSVersion: "TLSv1.3",
		}},
	}, {
		name: "realistic run with QUIC",
		args: args{
			begin: begin,
			events: []Event{&EventQUICHandshakeDone{&EventValue{
				Address:            "131.252.210.176:443",
				Err:                io.EOF,
				NoTLSVerify:        false,
				Proto:              "quic",
				TLSCipherSuite:     "SUITE",
				TLSNegotiatedProto: "h3",
				TLSPeerCerts: []*x509.Certificate{{
					Raw: []byte("deadbeef"),
				}, {
					Raw: []byte("abad1dea"),
				}},
				TLSServerName: "x.org",
				TLSVersion:    "TLSv1.3",
				Time:          begin.Add(55 * time.Millisecond),
			}}},
		},
		want: []TLSHandshake{{
			Address:            "131.252.210.176:443",
			CipherSuite:        "SUITE",
			Failure:            NewFailure(io.EOF),
			NegotiatedProtocol: "h3",
			NoTLSVerify:        false,
			PeerCertificates: []MaybeBinaryValue{{
				Value: "deadbeef",
			}, {
				Value: "abad1dea",
			}},
			ServerName: "x.org",
			T:          0.055,
			TLSVersion: "TLSv1.3",
		}},
	}, {
		name: "realistic run with no suitable events",
		args: args{
			begin: begin,
			events: []Event{&EventResolveStart{&EventValue{
				Time: begin.Add(55 * time.Millisecond),
			}}},
		},
		want: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewTLSHandshakesList(tt.args.begin, tt.args.events)
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNewFailure(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *string
	}{{
		name: "when error is nil",
		args: args{
			err: nil,
		},
		want: nil,
	}, {
		name: "when error is wrapped and failure meaningful",
		args: args{
			err: &netxlite.ErrWrapper{
				Failure: netxlite.FailureConnectionRefused,
			},
		},
		want: func() *string {
			s := netxlite.FailureConnectionRefused
			return &s
		}(),
	}, {
		name: "when error is wrapped and failure is not meaningful",
		args: args{
			err: &netxlite.ErrWrapper{},
		},
		want: func() *string {
			s := "unknown_failure: errWrapper.Failure is empty"
			return &s
		}(),
	}, {
		name: "when error is not wrapped but wrappable",
		args: args{err: io.EOF},
		want: func() *string {
			s := "eof_error"
			return &s
		}(),
	}, {
		name: "when the error is not wrapped and not wrappable",
		args: args{
			err: errors.New("use of closed socket 127.0.0.1:8080->10.0.0.1:22"),
		},
		want: func() *string {
			s := "unknown_failure: use of closed socket [scrubbed]->[scrubbed]"
			return &s
		}(),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFailure(tt.args.err)
			if tt.want == nil && got == nil {
				return
			}
			if tt.want == nil && got != nil {
				t.Errorf("NewFailure:  want %+v, got %s", tt.want, *got)
				return
			}
			if tt.want != nil && got == nil {
				t.Errorf("NewFailure:  want %s, got %+v", *tt.want, got)
				return
			}
			if *tt.want != *got {
				t.Errorf("NewFailure:  want %s, got %s", *tt.want, *got)
				return
			}
		})
	}
}

func TestNewFailedOperation(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want *string
	}{{
		name: "With no error",
		args: args{
			err: nil, // explicit
		},
		want: nil, // explicit
	}, {
		name: "With wrapped error and non-empty operation",
		args: args{
			err: &netxlite.ErrWrapper{
				Failure:   netxlite.FailureConnectionRefused,
				Operation: netxlite.ConnectOperation,
			},
		},
		want: (func() *string {
			s := netxlite.ConnectOperation
			return &s
		})(),
	}, {
		name: "With wrapped error and empty operation",
		args: args{
			err: &netxlite.ErrWrapper{
				Failure: netxlite.FailureConnectionRefused,
			},
		},
		want: (func() *string {
			s := netxlite.UnknownOperation
			return &s
		})(),
	}, {
		name: "With non wrapped error",
		args: args{
			err: io.EOF,
		},
		want: (func() *string {
			s := netxlite.UnknownOperation
			return &s
		})(),
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewFailedOperation(tt.args.err)
			if got == nil && tt.want == nil {
				return
			}
			if got == nil && tt.want != nil {
				t.Errorf("NewFailedOperation() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want == nil {
				t.Errorf("NewFailedOperation() = %v, want %v", got, tt.want)
				return
			}
			if got != nil && tt.want != nil && *got != *tt.want {
				t.Errorf("NewFailedOperation() = %v, want %v", got, tt.want)
				return
			}
		})
	}
}
