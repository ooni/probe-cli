package archival_test

import (
	"context"
	"crypto/x509"
	"errors"
	"io"
	"net/http"
	"reflect"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewTCPConnectList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []trace.Event
	}
	tests := []struct {
		name string
		args args
		want []archival.TCPConnectEntry
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
			events: []trace.Event{{
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "dns.google.com",
				Name:      "resolve_done",
				Time:      begin.Add(100 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Duration: 30 * time.Millisecond,
				Name:     netxlite.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Duration: 55 * time.Millisecond,
				Name:     netxlite.ConnectOperation,
				Proto:    "udp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Name:     netxlite.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(180 * time.Millisecond),
			}},
		},
		want: []archival.TCPConnectEntry{{
			IP:   "8.8.8.8",
			Port: 853,
			Status: archival.TCPConnectStatus{
				Success: true,
			},
			T: 0.13,
		}, {
			IP:   "8.8.4.4",
			Port: 53,
			Status: archival.TCPConnectStatus{
				Failure: archival.NewFailure(io.EOF),
				Success: false,
			},
			T: 0.18,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archival.NewTCPConnectList(tt.args.begin, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestNewRequestList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []trace.Event
	}
	tests := []struct {
		name string
		args args
		want []archival.RequestEntry
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
			events: []trace.Event{{
				Name: "http_transaction_start",
				Time: begin.Add(10 * time.Millisecond),
			}, {
				Name:            "http_request_body_snapshot",
				Data:            []byte("deadbeef"),
				DataIsTruncated: false,
			}, {
				Name: "http_request_metadata",
				HTTPHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "POST",
				HTTPURL:    "https://www.example.com/submit",
			}, {
				Name: "http_response_metadata",
				HTTPHeaders: http.Header{
					"Server": []string{"miniooni/0.1.0-dev"},
				},
				HTTPStatusCode: 200,
			}, {
				Name:            "http_response_body_snapshot",
				Data:            []byte("{}"),
				DataIsTruncated: false,
			}, {
				Name: "http_transaction_done",
			}, {
				Name: "http_transaction_start",
				Time: begin.Add(20 * time.Millisecond),
			}, {
				Name: "http_request_metadata",
				HTTPHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "GET",
				HTTPURL:    "https://www.example.com/result",
			}, {
				Name: "http_transaction_done",
				Err:  io.EOF,
			}},
		},
		want: []archival.RequestEntry{{
			Failure: archival.NewFailure(io.EOF),
			Request: archival.HTTPRequest{
				HeadersList: []archival.HTTPHeader{{
					Key: "User-Agent",
					Value: archival.MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "GET",
				URL:    "https://www.example.com/result",
			},
			T: 0.02,
		}, {
			Request: archival.HTTPRequest{
				Body: archival.MaybeBinaryValue{
					Value: "deadbeef",
				},
				HeadersList: []archival.HTTPHeader{{
					Key: "User-Agent",
					Value: archival.MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "POST",
				URL:    "https://www.example.com/submit",
			},
			Response: archival.HTTPResponse{
				Body: archival.MaybeBinaryValue{
					Value: "{}",
				},
				Code: 200,
				HeadersList: []archival.HTTPHeader{{
					Key: "Server",
					Value: archival.MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
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
			events: []trace.Event{{
				Name: "http_transaction_start",
				Time: begin.Add(10 * time.Millisecond),
			}, {
				Name: "http_request_metadata",
				HTTPHeaders: http.Header{
					"User-Agent": []string{"miniooni/0.1.0-dev"},
				},
				HTTPMethod: "GET",
				HTTPURL:    "https://www.example.com/",
			}, {
				Name: "http_response_metadata",
				HTTPHeaders: http.Header{
					"Server":   []string{"miniooni/0.1.0-dev"},
					"Location": []string{"https://x.example.com", "https://y.example.com"},
				},
				HTTPStatusCode: 302,
			}, {
				Name: "http_transaction_done",
			}},
		},
		want: []archival.RequestEntry{{
			Request: archival.HTTPRequest{
				HeadersList: []archival.HTTPHeader{{
					Key: "User-Agent",
					Value: archival.MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
					"User-Agent": {Value: "miniooni/0.1.0-dev"},
				},
				Method: "GET",
				URL:    "https://www.example.com/",
			},
			Response: archival.HTTPResponse{
				Code: 302,
				HeadersList: []archival.HTTPHeader{{
					Key: "Location",
					Value: archival.MaybeBinaryValue{
						Value: "https://x.example.com",
					},
				}, {
					Key: "Location",
					Value: archival.MaybeBinaryValue{
						Value: "https://y.example.com",
					},
				}, {
					Key: "Server",
					Value: archival.MaybeBinaryValue{
						Value: "miniooni/0.1.0-dev",
					},
				}},
				Headers: map[string]archival.MaybeBinaryValue{
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
			if got := archival.NewRequestList(tt.args.begin, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestNewDNSQueriesList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		events []trace.Event
	}
	tests := []struct {
		name string
		args args
		want []archival.DNSQueryEntry
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
			events: []trace.Event{{
				Address:   "1.1.1.1:853",
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
				Hostname:  "dns.google.com",
				Name:      "resolve_done",
				Proto:     "dot",
				Time:      begin.Add(100 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Duration: 30 * time.Millisecond,
				Name:     netxlite.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Name:     netxlite.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(180 * time.Millisecond),
			}},
		},
		want: []archival.DNSQueryEntry{{
			Answers: []archival.DNSAnswerEntry{{
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
			events: []trace.Event{{
				Addresses: []string{"2001:4860:4860::8888"},
				Hostname:  "dns.google.com",
				Name:      "resolve_done",
				Time:      begin.Add(200 * time.Millisecond),
			}},
		},
		want: []archival.DNSQueryEntry{{
			Answers: []archival.DNSAnswerEntry{{
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
			events: []trace.Event{{
				Err:      &netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError},
				Hostname: "dns.google.com",
				Name:     "resolve_done",
				Time:     begin.Add(200 * time.Millisecond),
			}},
		},
		want: []archival.DNSQueryEntry{{
			Answers: nil,
			Failure: archival.NewFailure(
				&netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError}),
			Hostname:  "dns.google.com",
			QueryType: "A",
			T:         0.2,
		}, {
			Answers: nil,
			Failure: archival.NewFailure(
				&netxlite.ErrWrapper{Failure: netxlite.FailureDNSNXDOMAINError}),
			Hostname:  "dns.google.com",
			QueryType: "AAAA",
			T:         0.2,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := archival.NewDNSQueriesList(tt.args.begin, tt.args.events)
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
		events []trace.Event
	}
	tests := []struct {
		name string
		args args
		want []archival.NetworkEvent
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
			events: []trace.Event{{
				Name:    netxlite.ConnectOperation,
				Address: "8.8.8.8:853",
				Err:     io.EOF,
				Proto:   "tcp",
				Time:    begin.Add(7 * time.Millisecond),
			}, {
				Name:     netxlite.ReadOperation,
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Name:     netxlite.ReadFromOperation,
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}, {
				Name:     netxlite.WriteOperation,
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Name:     netxlite.WriteToOperation,
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}, {
				Name: netxlite.CloseOperation,
				Err:  websocket.ErrReadLimit,
				Time: begin.Add(17 * time.Millisecond),
			}},
		},
		want: []archival.NetworkEvent{{
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(io.EOF),
			Operation: netxlite.ConnectOperation,
			Proto:     "tcp",
			T:         0.007,
		}, {
			Failure:   archival.NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: netxlite.ReadOperation,
			T:         0.011,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: netxlite.ReadFromOperation,
			T:         0.011,
		}, {
			Failure:   archival.NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: netxlite.WriteOperation,
			T:         0.014,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: netxlite.WriteToOperation,
			T:         0.014,
		}, {
			Failure:   archival.NewFailure(websocket.ErrReadLimit),
			Operation: netxlite.CloseOperation,
			T:         0.017,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archival.NewNetworkEventsList(tt.args.begin, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestNewTLSHandshakesList(t *testing.T) {
	begin := time.Now()
	type args struct {
		begin  time.Time
		target string
		events []trace.Event
	}
	tests := []struct {
		name string
		args args
		want []archival.TLSHandshake
	}{{
		name: "empty run",
		args: args{
			begin:  begin,
			target: "tlshandshake://0.0.0.0:443",
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin:  begin,
			target: "tlshandshake://131.252.210.176:443",
			events: []trace.Event{{
				Name: netxlite.CloseOperation,
				Err:  websocket.ErrReadLimit,
				Time: begin.Add(17 * time.Millisecond),
			}, {
				Name:               "tls_handshake_done",
				Err:                io.EOF,
				NoTLSVerify:        false,
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
			}},
		},
		want: []archival.TLSHandshake{{
			Address:            "131.252.210.176",
			CipherSuite:        "SUITE",
			Failure:            archival.NewFailure(io.EOF),
			NegotiatedProtocol: "h2",
			NoTLSVerify:        false,
			PeerCertificates: []archival.MaybeBinaryValue{{
				Value: "deadbeef",
			}, {
				Value: "abad1dea",
			}},
			ServerName: "x.org",
			T:          0.055,
			TLSVersion: "TLSv1.3",
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := archival.NewTLSHandshakesList(tt.args.begin, tt.args.target, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
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
			got := archival.NewFailure(tt.args.err)
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
			got := archival.NewFailedOperation(tt.args.err)
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
