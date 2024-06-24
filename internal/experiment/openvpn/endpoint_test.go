package openvpn

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
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
			name:    "empty input returns error",
			args:    args{""},
			want:    nil,
			wantErr: ErrInputRequired,
		},
		{
			name:    "invalid protocol returns error",
			args:    args{"bad://foo.bar"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "uri with illegal chars returns error",
			args:    args{"openvpn://\x7f/#"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name: "valid input uri returns good endpoint",
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

func Test_isValidProvider(t *testing.T) {
	if valid := isValidProvider("riseupvpn"); !valid {
		t.Fatal("riseup is the only valid provider now")
	}
	if valid := isValidProvider("nsa"); valid {
		t.Fatal("nsa will never be a provider")
	}
}

func Test_newVPNConfig(t *testing.T) {
	tracer := vpntracex.NewTracer(time.Now())
	e := &endpoint{
		Provider:  "riseupvpn",
		IPAddr:    "1.1.1.1",
		Port:      "443",
		Transport: "udp",
	}

	config := &Config{
		Auth:     "SHA512",
		Cipher:   "AES-256-GCM",
		SafeCA:   "ca",
		SafeCert: "cert",
		SafeKey:  "key",
	}

	cfg, err := newOpenVPNConfig(tracer, nil, e, config)
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
	if diff := cmp.Diff(cfg.OpenVPNOptions().CA, []byte(config.SafeCA)); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(cfg.OpenVPNOptions().Cert, []byte(config.SafeCert)); diff != "" {
		t.Error(diff)
	}
	if diff := cmp.Diff(cfg.OpenVPNOptions().Key, []byte(config.SafeKey)); diff != "" {
		t.Error(diff)
	}
}

func Test_mergeOpenVPNConfig_with_unknown_provider(t *testing.T) {
	tracer := vpntracex.NewTracer(time.Now())
	e := &endpoint{
		Provider:  "nsa",
		IPAddr:    "1.1.1.1",
		Port:      "443",
		Transport: "udp",
	}
	cfg := &Config{
		SafeCA:   "ca",
		SafeCert: "cert",
		SafeKey:  "key",
	}
	_, err := newOpenVPNConfig(tracer, nil, e, cfg)
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input error, got: %v", err)
	}
}
