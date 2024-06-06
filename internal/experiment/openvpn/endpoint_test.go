package openvpn

import (
	"errors"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

func Test_newEndpointFromInputString(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    *endpoint
		wantErr error
	}{
		{
			name: "valid endpoint returns good endpoint",
			args: args{"openvpn://riseupvpn.corp/?address=1.1.1.1:1194&transport=tcp"},
			want: &endpoint{
				IPAddr:      "1.1.1.1",
				Obfuscation: "none",
				Port:        "1194",
				Protocol:    "openvpn",
				Provider:    "riseupvpn",
				Transport:   "tcp",
			},
			wantErr: nil,
		},
		{
			name:    "bad url fails",
			args:    args{"://address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name: "openvpn+obfs4 does not fail",
			args: args{"openvpn+obfs4://riseupvpn.corp/?address=1.1.1.1:1194&transport=tcp"},
			want: &endpoint{
				IPAddr:      "1.1.1.1",
				Obfuscation: "obfs4",
				Port:        "1194",
				Protocol:    "openvpn",
				Provider:    "riseupvpn",
				Transport:   "tcp",
			},
			wantErr: nil,
		},
		{
			name:    "unknown proto fails",
			args:    args{"unknown://riseupvpn.corp/?address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "any tld other than .corp fails",
			args:    args{"openvpn://riseupvpn.org/?address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "empty provider fails",
			args:    args{"openvpn://.corp/?address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "non-registered provider fails",
			args:    args{"openvpn://nsavpn.corp/?address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with invalid ipv4 fails",
			args:    args{"openvpn://riseupvpn.corp/?address=example.com:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with no port fails",
			args:    args{"openvpn://riseupvpn.corp/?address=1.1.1.1&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with empty transport fails",
			args:    args{"openvpn://riseupvpn.corp/?address=1.1.1.1:1194&transport="},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with no transport fails",
			args:    args{"openvpn://riseupvpn.corp/?address=1.1.1.1:1194"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with unknown transport fails",
			args:    args{"openvpn://riseupvpn.corp/?address=1.1.1.1:1194&transport=uh"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with no address fails",
			args:    args{"openvpn://riseupvpn.corp/?transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with empty address fails",
			args:    args{"openvpn://riseupvpn.corp/?address=&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := newEndpointFromInputString(tt.args.uri)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("newEndpointFromInputString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if diff := cmp.Diff(got, tt.want); diff != "" {
				t.Errorf(diff)
			}
		})
	}
}

func Test_EndpointToInputURI(t *testing.T) {
	type args struct {
		endpoint endpoint
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "good endpoint with plain openvpn",
			args: args{
				endpoint{
					IPAddr:      "1.1.1.1",
					Obfuscation: "none",
					Port:        "443",
					Protocol:    "openvpn",
					Provider:    "shady",
					Transport:   "udp",
				},
			},
			want: "openvpn://shady.corp?address=1.1.1.1%3A443&transport=udp",
		},
		{
			name: "good endpoint with openvpn+obfs4",
			args: args{
				endpoint{
					IPAddr:      "1.1.1.1",
					Obfuscation: "obfs4",
					Port:        "443",
					Protocol:    "openvpn",
					Provider:    "shady",
					Transport:   "udp",
				},
			},
			want: "openvpn+obfs4://shady.corp?address=1.1.1.1%3A443&transport=udp",
		},
		{
			name: "empty provider is marked as unknown",
			args: args{
				endpoint{
					IPAddr:      "1.1.1.1",
					Obfuscation: "obfs4",
					Port:        "443",
					Protocol:    "openvpn",
					Provider:    "",
					Transport:   "udp",
				},
			},
			want: "openvpn+obfs4://unknown.corp?address=1.1.1.1%3A443&transport=udp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.endpoint.AsInputURI(); cmp.Diff(got, tt.want) != "" {
				fmt.Println("GOT", got)
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

func Test_endpoint_String(t *testing.T) {
	type fields struct {
		IPAddr      string
		Obfuscation string
		Port        string
		Protocol    string
		Provider    string
		Transport   string
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "well formed endpoint returns a well formed endpoint string",
			fields: fields{
				IPAddr:      "1.1.1.1",
				Obfuscation: "none",
				Port:        "1194",
				Protocol:    "openvpn",
				Provider:    "unknown",
				Transport:   "tcp",
			},
			want: "openvpn://1.1.1.1:1194/tcp",
		},
		{
			name: "well formed endpoint, openvpn+obfs4",
			fields: fields{
				IPAddr:      "1.1.1.1",
				Obfuscation: "obfs4",
				Port:        "1194",
				Protocol:    "openvpn",
				Provider:    "unknown",
				Transport:   "tcp",
			},
			want: "openvpn+obfs4://1.1.1.1:1194/tcp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &endpoint{
				IPAddr:      tt.fields.IPAddr,
				Obfuscation: tt.fields.Obfuscation,
				Port:        tt.fields.Port,
				Protocol:    tt.fields.Protocol,
				Provider:    tt.fields.Provider,
				Transport:   tt.fields.Transport,
			}
			if got := e.String(); got != tt.want {
				t.Errorf("endpoint.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_endpointList_Shuffle(t *testing.T) {
	shuffled := DefaultEndpoints.Shuffle()
	sort.Slice(shuffled, func(i, j int) bool {
		return shuffled[i].IPAddr < shuffled[j].IPAddr
	})
	if diff := cmp.Diff(shuffled, DefaultEndpoints); diff != "" {
		t.Error(diff)
	}
}

func Test_isValidProvider(t *testing.T) {
	if valid := isValidProvider("riseupvpn"); !valid {
		t.Fatal("riseup is the only valid provider now")
	}
	if valid := isValidProvider("nsa"); valid {
		t.Fatal("nsa will never be a provider")
	}
}

func Test_getVPNConfig(t *testing.T) {
	tracer := vpntracex.NewTracer(time.Now())
	e := &endpoint{
		Provider:  "riseupvpn",
		IPAddr:    "1.1.1.1",
		Port:      "443",
		Transport: "udp",
	}
	creds := &vpnconfig.OpenVPNOptions{
		CA:   []byte("ca"),
		Cert: []byte("cert"),
		Key:  []byte("key"),
	}

	cfg, err := getOpenVPNConfig(tracer, nil, e, creds)
	if err != nil {
		t.Fatalf("did not expect error, got: %v", err)
	}
	if cfg.Tracer() != tracer {
		t.Fatal("config tracer is not what passed")
	}
	if auth := cfg.OpenVPNOptions().Auth; auth != "SHA512" {
		t.Errorf("expected auth %s, got %s", "SHA512", auth)
	}
	if cipher := cfg.OpenVPNOptions().Cipher; cipher != "AES-256-GCM" {
		t.Errorf("expected cipher %s, got %s", "AES-256-GCM", cipher)
	}
	if remote := cfg.OpenVPNOptions().Remote; remote != e.IPAddr {
		t.Errorf("expected remote %s, got %s", e.IPAddr, remote)
	}
	if port := cfg.OpenVPNOptions().Port; port != e.Port {
		t.Errorf("expected port %s, got %s", e.Port, port)
	}
	if transport := cfg.OpenVPNOptions().Proto; string(transport) != e.Transport {
		t.Errorf("expected transport %s, got %s", e.Transport, transport)
	}
	if transport := cfg.OpenVPNOptions().Proto; string(transport) != e.Transport {
		t.Errorf("expected transport %s, got %s", e.Transport, transport)
	}
	if diff := cmp.Diff(cfg.OpenVPNOptions().CA, creds.CA); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(cfg.OpenVPNOptions().Cert, creds.Cert); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(cfg.OpenVPNOptions().Key, creds.Key); diff != "" {
		t.Error(diff)
	}
}

func Test_getVPNConfig_with_unknown_provider(t *testing.T) {
	tracer := vpntracex.NewTracer(time.Now())
	e := &endpoint{
		Provider:  "nsa",
		IPAddr:    "1.1.1.1",
		Port:      "443",
		Transport: "udp",
	}
	creds := &vpnconfig.OpenVPNOptions{
		CA:   []byte("ca"),
		Cert: []byte("cert"),
		Key:  []byte("key"),
	}
	_, err := getOpenVPNConfig(tracer, nil, e, creds)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got: %v", err)
	}

}

func Test_extractBase64Blob(t *testing.T) {
	t.Run("decode good blob", func(t *testing.T) {
		blob := "base64:dGhlIGJsdWUgb2N0b3B1cyBpcyB3YXRjaGluZw=="
		decoded, err := maybeExtractBase64Blob(blob)
		if decoded != "the blue octopus is watching" {
			t.Fatal("could not decoded blob correctly")
		}
		if err != nil {
			t.Fatal("should not fail with first blob")
		}
	})
	t.Run("try decode without prefix", func(t *testing.T) {
		blob := "dGhlIGJsdWUgb2N0b3B1cyBpcyB3YXRjaGluZw=="
		dec, err := maybeExtractBase64Blob(blob)
		if err != nil {
			t.Fatal("should fail without prefix")
		}
		if dec != blob {
			t.Fatal("decoded should be the same")
		}
	})
	t.Run("bad base64 blob should fail", func(t *testing.T) {
		blob := "base64:dGhlIGJsdWUgb2N0b3B1cyBpcyB3YXRjaGluZw"
		_, err := maybeExtractBase64Blob(blob)
		if !errors.Is(err, ErrBadBase64Blob) {
			t.Fatal("bad blob should fail without prefix")
		}
	})
	t.Run("decode empty blob", func(t *testing.T) {
		blob := "base64:"
		_, err := maybeExtractBase64Blob(blob)
		if err != nil {
			t.Fatal("empty blob should not fail")
		}
	})
	t.Run("illegal base64 data should fail", func(t *testing.T) {
		blob := "base64:=="
		_, err := maybeExtractBase64Blob(blob)
		if !errors.Is(err, ErrBadBase64Blob) {
			t.Fatal("bad base64 data should fail")
		}
	})
}

func Test_IsValidProtocol(t *testing.T) {
	t.Run("openvpn is valid", func(t *testing.T) {
		if !isValidProtocol("openvpn://foobar.bar") {
			t.Error("openvpn:// should be a valid protocol")
		}
	})
	t.Run("openvpn+obfs4 is valid", func(t *testing.T) {
		if !isValidProtocol("openvpn+obfs4://foobar.bar") {
			t.Error("openvpn+obfs4:// should be a valid protocol")
		}
	})
	t.Run("openvpn+other is not valid", func(t *testing.T) {
		if isValidProtocol("openvpn+ss://foobar.bar") {
			t.Error("openvpn+ss:// should not be a valid protocol")
		}
	})
}
