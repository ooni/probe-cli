package openvpn

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
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
			args: args{"openvpn://riseup.corp/?address=1.1.1.1:1194&transport=tcp"},
			want: &endpoint{
				IPAddr:      "1.1.1.1",
				Obfuscation: "none",
				Port:        "1194",
				Protocol:    "openvpn",
				Provider:    "riseup",
				Transport:   "tcp",
			},
			wantErr: nil,
		},
		{
			name:    "unknown proto fails",
			args:    args{"unknown://riseup.corp/?address=1.1.1.1:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "any tld other than .corp fails",
			args:    args{"openvpn://riseup.org/?address=1.1.1.1:1194&transport=tcp"},
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
			args:    args{"openvpn://riseup.corp/?address=example.com:1194&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with no port fails",
			args:    args{"openvpn://riseup.corp/?address=1.1.1.1&transport=tcp"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with empty transport fails",
			args:    args{"openvpn://riseup.corp/?address=1.1.1.1:1194&transport="},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with no transport fails",
			args:    args{"openvpn://riseup.corp/?address=1.1.1.1:1194"},
			want:    nil,
			wantErr: ErrInvalidInput,
		},
		{
			name:    "endpoint with unknown transport fails",
			args:    args{"openvpn://riseup.corp/?address=1.1.1.1:1194&transport=uh"},
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
			want: "openvpn://shady.corp/?address=1.1.1.1:443&transport=udp",
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
			want: "openvpn+obfs4://shady.corp/?address=1.1.1.1:443&transport=udp",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.args.endpoint.AsInputURI(); cmp.Diff(got, tt.want) != "" {
				t.Error(cmp.Diff(got, tt.want))
			}
		})
	}
}

// TODO: test the endpoint uri string too.
