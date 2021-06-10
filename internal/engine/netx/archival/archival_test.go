package archival_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"io"
	"net"
	"net/http"
	"reflect"
	"syscall"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/gorilla/websocket"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/resolver"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/pion/stun"
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
				Name:     errorx.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Duration: 55 * time.Millisecond,
				Name:     errorx.ConnectOperation,
				Proto:    "udp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Name:     errorx.ConnectOperation,
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
		// https://github.com/ooni/probe-cli/v3/internal/engine/pull/751/checks?check_run_id=853562310
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
				Name:     errorx.ConnectOperation,
				Proto:    "tcp",
				Time:     begin.Add(130 * time.Millisecond),
			}, {
				Address:  "8.8.4.4:53",
				Duration: 50 * time.Millisecond,
				Err:      io.EOF,
				Name:     errorx.ConnectOperation,
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
				Err:      dialer.NewErrDial(errorx.ErrDNSBogon),
				Hostname: "dns.google.com",
				Name:     "resolve_done",
				Time:     begin.Add(200 * time.Millisecond),
			}},
		},
		want: []archival.DNSQueryEntry{{
			Answers: nil,
			Failure: archival.NewFailure(
				dialer.NewErrDial(errorx.ErrDNSBogon)),
			Hostname:  "dns.google.com",
			QueryType: "A",
			T:         0.2,
		}, {
			Answers: nil,
			Failure: archival.NewFailure(
				dialer.NewErrDial(errorx.ErrDNSBogon)),
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
				Name:    errorx.ConnectOperation,
				Address: "8.8.8.8:853",
				Err:     io.EOF,
				Proto:   "tcp",
				Time:    begin.Add(7 * time.Millisecond),
			}, {
				Name:     errorx.ReadOperation,
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Name:     errorx.ReadFromOperation,
				Err:      context.Canceled,
				NumBytes: 7117,
				Time:     begin.Add(11 * time.Millisecond),
			}, {
				Name:     errorx.WriteOperation,
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}, {
				Address:  "8.8.8.8:853",
				Name:     errorx.WriteToOperation,
				Err:      websocket.ErrBadHandshake,
				NumBytes: 4114,
				Time:     begin.Add(14 * time.Millisecond),
			}, {
				Name: errorx.CloseOperation,
				Err:  websocket.ErrReadLimit,
				Time: begin.Add(17 * time.Millisecond),
			}},
		},
		want: []archival.NetworkEvent{{
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(io.EOF),
			Operation: errorx.ConnectOperation,
			Proto:     "tcp",
			T:         0.007,
		}, {
			Failure:   archival.NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: errorx.ReadOperation,
			T:         0.011,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(context.Canceled),
			NumBytes:  7117,
			Operation: errorx.ReadFromOperation,
			T:         0.011,
		}, {
			Failure:   archival.NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: errorx.WriteOperation,
			T:         0.014,
		}, {
			Address:   "8.8.8.8:853",
			Failure:   archival.NewFailure(websocket.ErrBadHandshake),
			NumBytes:  4114,
			Operation: errorx.WriteToOperation,
			T:         0.014,
		}, {
			Failure:   archival.NewFailure(websocket.ErrReadLimit),
			Operation: errorx.CloseOperation,
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
			events: nil,
		},
		want: nil,
	}, {
		name: "realistic run",
		args: args{
			begin: begin,
			events: []trace.Event{{
				Name: errorx.CloseOperation,
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
			if got := archival.NewTLSHandshakesList(tt.args.begin, tt.args.events); !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestExtSpec_AddTo(t *testing.T) {
	m := new(model.Measurement)
	archival.ExtDNS.AddTo(m)
	expected := map[string]int64{"dnst": 0}
	if d := cmp.Diff(m.Extensions, expected); d != "" {
		t.Fatal(d)
	}
}

var binaryInput = []uint8{
	0x57, 0xe5, 0x79, 0xfb, 0xa6, 0xbb, 0x0d, 0xbc, 0xce, 0xbd, 0xa7, 0xa0,
	0xba, 0xa4, 0x78, 0x78, 0x12, 0x59, 0xee, 0x68, 0x39, 0xa4, 0x07, 0x98,
	0xc5, 0x3e, 0xbc, 0x55, 0xcb, 0xfe, 0x34, 0x3c, 0x7e, 0x1b, 0x5a, 0xb3,
	0x22, 0x9d, 0xc1, 0x2d, 0x6e, 0xca, 0x5b, 0xf1, 0x10, 0x25, 0x47, 0x1e,
	0x44, 0xe2, 0x2d, 0x60, 0x08, 0xea, 0xb0, 0x0a, 0xcc, 0x05, 0x48, 0xa0,
	0xf5, 0x78, 0x38, 0xf0, 0xdb, 0x3f, 0x9d, 0x9f, 0x25, 0x6f, 0x89, 0x00,
	0x96, 0x93, 0xaf, 0x43, 0xac, 0x4d, 0xc9, 0xac, 0x13, 0xdb, 0x22, 0xbe,
	0x7a, 0x7d, 0xd9, 0x24, 0xa2, 0x52, 0x69, 0xd8, 0x89, 0xc1, 0xd1, 0x57,
	0xaa, 0x04, 0x2b, 0xa2, 0xd8, 0xb1, 0x19, 0xf6, 0xd5, 0x11, 0x39, 0xbb,
	0x80, 0xcf, 0x86, 0xf9, 0x5f, 0x9d, 0x8c, 0xab, 0xf5, 0xc5, 0x74, 0x24,
	0x3a, 0xa2, 0xd4, 0x40, 0x4e, 0xd7, 0x10, 0x1f,
}

var encodedBinaryInput = []byte(`{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}`)

func TestMaybeBinaryValue_MarshalJSON(t *testing.T) {
	type fields struct {
		Value string
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{{
		name: "with string input",
		fields: fields{
			Value: "antani",
		},
		want:    []byte(`"antani"`),
		wantErr: false,
	}, {
		name: "with binary input",
		fields: fields{
			Value: string(binaryInput),
		},
		want:    encodedBinaryInput,
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hb := archival.MaybeBinaryValue{
				Value: tt.fields.Value,
			}
			got, err := hb.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("MaybeBinaryValue.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestMaybeBinaryValue_UnmarshalJSON(t *testing.T) {
	type fields struct {
		WantValue string
	}
	type args struct {
		d []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{{
		name: "with string input",
		fields: fields{
			WantValue: "xo",
		},
		args:    args{d: []byte(`"xo"`)},
		wantErr: false,
	}, {
		name: "with nil input",
		fields: fields{
			WantValue: "",
		},
		args:    args{d: nil},
		wantErr: true,
	}, {
		name: "with missing/invalid format",
		fields: fields{
			WantValue: "",
		},
		args:    args{d: []byte(`{"format": "foo"}`)},
		wantErr: true,
	}, {
		name: "with missing data",
		fields: fields{
			WantValue: "",
		},
		args:    args{d: []byte(`{"format": "base64"}`)},
		wantErr: true,
	}, {
		name: "with invalid base64 data",
		fields: fields{
			WantValue: "",
		},
		args:    args{d: []byte(`{"format": "base64", "data": "x"}`)},
		wantErr: true,
	}, {
		name: "with valid base64 data",
		fields: fields{
			WantValue: string(binaryInput),
		},
		args:    args{d: encodedBinaryInput},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hb := &archival.MaybeBinaryValue{}
			if err := hb.UnmarshalJSON(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("MaybeBinaryValue.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			if d := cmp.Diff(tt.fields.WantValue, hb.Value); d != "" {
				t.Error(d)
			}
		})
	}
}

func TestHTTPHeader_MarshalJSON(t *testing.T) {
	type fields struct {
		Key   string
		Value archival.MaybeBinaryValue
	}
	tests := []struct {
		name    string
		fields  fields
		want    []byte
		wantErr bool
	}{{
		name: "with string value",
		fields: fields{
			Key: "Content-Type",
			Value: archival.MaybeBinaryValue{
				Value: "text/plain",
			},
		},
		want:    []byte(`["Content-Type","text/plain"]`),
		wantErr: false,
	}, {
		name: "with binary value",
		fields: fields{
			Key: "Content-Type",
			Value: archival.MaybeBinaryValue{
				Value: string(binaryInput),
			},
		},
		want:    []byte(`["Content-Type",` + string(encodedBinaryInput) + `]`),
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hh := archival.HTTPHeader{
				Key:   tt.fields.Key,
				Value: tt.fields.Value,
			}
			got, err := hh.MarshalJSON()
			if (err != nil) != tt.wantErr {
				t.Errorf("HTTPHeader.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func TestHTTPHeader_UnmarshalJSON(t *testing.T) {
	type fields struct {
		WantKey   string
		WantValue archival.MaybeBinaryValue
	}
	type args struct {
		d []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{{
		name: "with invalid input",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`{}`),
		},
		wantErr: true,
	}, {
		name: "with unexpected number of items",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`[]`),
		},
		wantErr: true,
	}, {
		name: "with first item not being a string",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`[0,0]`),
		},
		wantErr: true,
	}, {
		name: "with both items being a string",
		fields: fields{
			WantKey: "x",
			WantValue: archival.MaybeBinaryValue{
				Value: "y",
			},
		},
		args: args{
			d: []byte(`["x","y"]`),
		},
		wantErr: false,
	}, {
		name: "with second item not being a map[string]interface{}",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",[]]`),
		},
		wantErr: true,
	}, {
		name: "with missing format key in second item",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",{}]`),
		},
		wantErr: true,
	}, {
		name: "with format value not being base64",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",{"format":1}]`),
		},
		wantErr: true,
	}, {
		name: "with missing data field",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",{"format":"base64"}]`),
		},
		wantErr: true,
	}, {
		name: "with data not being a string",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",{"format":"base64","data":1}]`),
		},
		wantErr: true,
	}, {
		name: "with data not being base64",
		fields: fields{
			WantKey: "",
			WantValue: archival.MaybeBinaryValue{
				Value: "",
			},
		},
		args: args{
			d: []byte(`["x",{"format":"base64","data":"xx"}]`),
		},
		wantErr: true,
	}, {
		name: "with correctly encoded base64 data",
		fields: fields{
			WantKey: "x",
			WantValue: archival.MaybeBinaryValue{
				Value: string(binaryInput),
			},
		},
		args: args{
			d: []byte(`["x",` + string(encodedBinaryInput) + `]`),
		},
		wantErr: false,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hh := &archival.HTTPHeader{}
			if err := hh.UnmarshalJSON(tt.args.d); (err != nil) != tt.wantErr {
				t.Errorf("HTTPHeader.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
			}
			expect := &archival.HTTPHeader{
				Key:   tt.fields.WantKey,
				Value: tt.fields.WantValue,
			}
			if d := cmp.Diff(hh, expect); d != "" {
				t.Error(d)
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
			err: dialer.MockErrDial,
		},
		want: func() *string {
			s := "unknown_failure: mock error"
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
		name: "With ErrDial",
		args: args{
			err: dialer.MockErrDial,
		},
		want: (func() *string {
			s := errorx.ConnectOperation
			return &s
		})(),
	}, {
		name: "With ErrWrite",
		args: args{
			err: dialer.MockErrWrite,
		},
		want: (func() *string {
			s := errorx.WriteOperation
			return &s
		})(),
	}, {
		name: "With ErrRead",
		args: args{
			err: dialer.MockErrRead,
		},
		want: (func() *string {
			s := errorx.ReadOperation
			return &s
		})(),
	}, {
		name: "With ErrTLSHandshake",
		args: args{
			err: tlsdialer.MockErrTLSHandshake,
		},
		want: (func() *string {
			s := errorx.TLSHandshakeOperation
			return &s
		})(),
	}, {
		name: "With ErrClose",
		args: args{
			err: dialer.MockErrClose,
		},
		want: (func() *string {
			s := errorx.CloseOperation
			return &s
		})(),
	}, {
		name: "With ErrReadFrom",
		args: args{
			err: quicdialer.MockErrReadFrom,
		},
		want: (func() *string {
			s := errorx.ReadFromOperation
			return &s
		})(),
	}, {
		name: "With ErrWriteTo",
		args: args{
			err: quicdialer.MockErrWriteTo,
		},
		want: (func() *string {
			s := errorx.WriteToOperation
			return &s
		})(),
	},
		{
			name: "With ErrResolve",
			args: args{
				err: resolver.MockErrResolve,
			},
			want: (func() *string {
				s := errorx.ResolveOperation
				return &s
			})(),
		}, {
			name: "With top level operation",
			args: args{
				err: io.EOF,
			},
			want: (func() *string {
				s := errorx.TopLevelOperation
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

func TestToFailureString(t *testing.T) {
	t.Run("for ErrDNSBogon", func(t *testing.T) {
		if archival.ToFailureString(errorx.ErrDNSBogon) != errorx.FailureDNSBogonError {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for context.Canceled", func(t *testing.T) {
		if archival.ToFailureString(context.Canceled) != errorx.FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.HostnameError", func(t *testing.T) {
		var err x509.HostnameError
		if archival.ToFailureString(err) != errorx.FailureSSLInvalidHostname {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.UnknownAuthorityError", func(t *testing.T) {
		var err x509.UnknownAuthorityError
		if archival.ToFailureString(err) != errorx.FailureSSLUnknownAuthority {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for x509.CertificateInvalidError", func(t *testing.T) {
		var err x509.CertificateInvalidError
		if archival.ToFailureString(err) != errorx.FailureSSLInvalidCertificate {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for operation was canceled error", func(t *testing.T) {
		if archival.ToFailureString(errors.New("operation was canceled")) != errorx.FailureInterrupted {
			t.Fatal("unexpected result")
		}
	})
	t.Run("for EOF", func(t *testing.T) {
		if archival.ToFailureString(io.EOF) != errorx.FailureEOFError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_refused", func(t *testing.T) {
		if archival.ToFailureString(syscall.ECONNREFUSED) != errorx.FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if archival.ToFailureString(syscall.ECONNRESET) != errorx.FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset wrapped", func(t *testing.T) {
		if archival.ToFailureString(&net.OpError{Err: syscall.ECONNRESET}) != errorx.FailureConnectionReset {
			t.Fatal("unexpected results", archival.ToFailureString(&net.DNSConfigError{Err: syscall.ECONNRESET}))
		}
	})
	t.Run("for context deadline exceeded", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel()
		<-ctx.Done()
		if archival.ToFailureString(ctx.Err()) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for stun's transaction is timed out", func(t *testing.T) {
		if archival.ToFailureString(stun.ErrTransactionTimeOut) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for i/o error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel() // fail immediately
		conn, err := (&net.Dialer{}).DialContext(ctx, "tcp", "www.google.com:80")
		if err == nil {
			t.Fatal("expected an error here")
		}
		if conn != nil {
			t.Fatal("expected nil connection here")
		}
		if archival.ToFailureString(err) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for TLS handshake timeout error", func(t *testing.T) {
		err := errors.New("net/http: TLS handshake timeout")
		if archival.ToFailureString(err) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for no such host", func(t *testing.T) {
		if archival.ToFailureString(&net.DNSError{
			Err: "no such host",
		}) != errorx.FailureDNSNXDOMAINError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for errors including IPv4 address", func(t *testing.T) {
		input := errors.New("read tcp 10.0.2.15:56948->93.184.216.34:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := archival.ToFailureString(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	t.Run("for errors including IPv6 address", func(t *testing.T) {
		input := errors.New("read tcp [::1]:56948->[::1]:443: use of closed network connection")
		expected := "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection"
		out := archival.ToFailureString(input)
		if out != expected {
			t.Fatal(cmp.Diff(expected, out))
		}
	})
	// QUIC failures
	t.Run("for connection_refused", func(t *testing.T) {
		if archival.ToFailureString(&quic.TransportError{ErrorCode: quic.ConnectionRefused}) != errorx.FailureConnectionRefused {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for connection_reset", func(t *testing.T) {
		if archival.ToFailureString(&quic.StatelessResetError{}) != errorx.FailureConnectionReset {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for incompatible quic version", func(t *testing.T) {
		if archival.ToFailureString(&quic.VersionNegotiationError{}) != errorx.FailureNoCompatibleQUICVersion {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for i/o error", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1)
		defer cancel() // fail immediately
		udpAddr := &net.UDPAddr{IP: net.ParseIP("216.58.212.164"), Port: 80, Zone: ""}
		udpConn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
		if err != nil {
			t.Fatal(err)
		}
		sess, err := quic.DialEarlyContext(ctx, udpConn, udpAddr, "google.com:80", &tls.Config{}, &quic.Config{})
		if err == nil {
			t.Fatal("expected an error here")
		}
		if sess != nil {
			t.Fatal("expected nil session here")
		}
		if archival.ToFailureString(err) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC handshake timeout error", func(t *testing.T) {
		err := &quic.HandshakeTimeoutError{}
		if archival.ToFailureString(err) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
	t.Run("for QUIC connection timeout error", func(t *testing.T) {
		err := &quic.IdleTimeoutError{}
		if archival.ToFailureString(err) != errorx.FailureGenericTimeoutError {
			t.Fatal("unexpected results")
		}
	})
}
