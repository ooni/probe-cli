package legacymodel_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/legacy/legacymodel"
)

// we use this value below to test we can handle binary data
var archivalBinaryInput = []uint8{
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

// we use this value below to test we can handle binary data
var archivalEncodedBinaryInput = []byte(`{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}`)

func TestArchivalMaybeBinaryData(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		tests := []struct {
			name    string // test name
			input   string // value to marshal
			want    []byte // expected result
			wantErr bool   // whether we expect an error
		}{{
			name:    "with string input",
			input:   "antani",
			want:    []byte(`"antani"`),
			wantErr: false,
		}, {
			name:    "with binary input",
			input:   string(archivalBinaryInput),
			want:    archivalEncodedBinaryInput,
			wantErr: false,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hb := legacymodel.ArchivalMaybeBinaryData{
					Value: tt.input,
				}
				got, err := hb.MarshalJSON()
				if (err != nil) != tt.wantErr {
					t.Fatalf("ArchivalMaybeBinaryData.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			name    string // test name
			input   []byte // value to unmarshal
			want    string // expected result
			wantErr bool   // whether we want an error
		}{{
			name:    "with string input",
			input:   []byte(`"xo"`),
			want:    "xo",
			wantErr: false,
		}, {
			name:    "with nil input",
			input:   nil,
			want:    "",
			wantErr: true,
		}, {
			name:    "with missing/invalid format",
			input:   []byte(`{"format": "foo"}`),
			want:    "",
			wantErr: true,
		}, {
			name:    "with missing data",
			input:   []byte(`{"format": "base64"}`),
			want:    "",
			wantErr: true,
		}, {
			name:    "with invalid base64 data",
			input:   []byte(`{"format": "base64", "data": "x"}`),
			want:    "",
			wantErr: true,
		}, {
			name:    "with valid base64 data",
			input:   archivalEncodedBinaryInput,
			want:    string(archivalBinaryInput),
			wantErr: false,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hb := &legacymodel.ArchivalMaybeBinaryData{}
				if err := hb.UnmarshalJSON(tt.input); (err != nil) != tt.wantErr {
					t.Fatalf("ArchivalMaybeBinaryData.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				if d := cmp.Diff(tt.want, hb.Value); d != "" {
					t.Fatal(d)
				}
			})
		}
	})
}
