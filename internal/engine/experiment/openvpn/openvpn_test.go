package openvpn

import (
	"errors"
	"reflect"
	"testing"
)

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "torsf" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.2.0" {
		t.Fatal("invalid experiment version")
	}
}

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
			args: args{"openvpn://aprovider@10.0.0.1:9000/udp/?obfs=none"},
			want: &VPNExperiment{
				Provider:    "aprovider",
				Address:     "10.0.0.1:9000",
				Protocol:    "openvpn",
				Transport:   "udp",
				Obfuscation: "none",
			},
			wantErr: nil,
		},
		{
			name: "no provider is unknown provider",
			args: args{"openvpn://10.0.0.1:9000/udp/?obfs=none"},
			want: &VPNExperiment{
				Provider:    "unknown",
				Address:     "10.0.0.1:9000",
				Protocol:    "openvpn",
				Transport:   "udp",
				Obfuscation: "none",
			},
			wantErr: nil,
		},
		{
			name:    "unknown vpn protocol handler raises error",
			args:    args{"novpn://10.0.0.1:9000/udp/?obfs=none"},
			want:    &VPNExperiment{},
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
