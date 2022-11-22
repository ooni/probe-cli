package openvpn

import (
	"errors"
	"reflect"
	"testing"
)

func Test_vpnExperimentFromURI(t *testing.T) {
	type args struct {
		uri string
	}
	tests := []struct {
		name    string
		args    args
		want    *VPNExperiment
		wantErr error
	}{
		{
			name: "sunny path does not fail",
			args: args{"vpn://aprovider.openvpn/?addr=10.0.0.1:9000&transport=udp&obfuscation=none"},
			want: &VPNExperiment{
				Provider:    "aprovider",
				Hostname:    "10.0.0.1",
				Port:        "9000",
				Protocol:    "openvpn",
				Transport:   "udp",
				Obfuscation: "none",
			},
			wantErr: nil,
		},
		{
			name: "no provider is unknown provider",
			args: args{"vpn://openvpn/?addr=10.0.0.1:9000&transport=udp&obfuscation=none"},
			want: &VPNExperiment{
				Provider:    "unknown",
				Hostname:    "10.0.0.1",
				Port:        "9000",
				Protocol:    "openvpn",
				Transport:   "udp",
				Obfuscation: "none",
			},
			wantErr: nil,
		},
		{
			name: "no obfuscation is obsfuscation = none",
			args: args{"vpn://aprovider.openvpn/?addr=10.0.0.1:9000&transport=udp"},
			want: &VPNExperiment{
				Provider:    "aprovider",
				Hostname:    "10.0.0.1",
				Port:        "9000",
				Protocol:    "openvpn",
				Transport:   "udp",
				Obfuscation: "none",
			},
			wantErr: nil,
		},
		{
			name:    "no transport raises error",
			args:    args{"vpn://aprovider.openvpn/?addr=10.0.0.1:9000"},
			want:    nil,
			wantErr: BadOONIRunInput,
		},
		{
			name:    "unknown vpn protocol handler raises error",
			args:    args{"novpn://foo/?transport=udp"},
			want:    nil,
			wantErr: BadOONIRunInput,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := vpnExperimentFromURI(tt.args.uri)
			if !errors.Is(err, tt.wantErr) {
				t.Errorf("vpnExperimentFromURI() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("vpnExperimentFromURI() = %v, want %v", got, tt.want)
			}
		})
	}
}
