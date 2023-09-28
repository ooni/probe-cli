package model_test

import (
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestArchivalExtSpec(t *testing.T) {
	t.Run("AddTo", func(t *testing.T) {
		m := &model.Measurement{}
		model.ArchivalExtDNS.AddTo(m)
		expected := map[string]int64{"dnst": 0}
		if d := cmp.Diff(m.Extensions, expected); d != "" {
			t.Fatal(d)
		}
	})
}

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

func TestArchivalBinaryData(t *testing.T) {
	// This test verifies that we correctly serialize binary data to JSON by
	// producing null | {"format":"base64","data":"<base64>"}
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input model.ArchivalBinaryData

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name:       "with nil .Value",
			input:      model.ArchivalBinaryData{Value: nil},
			expectErr:  nil,
			expectData: []byte("null"),
		}, {
			name:       "with zero length .Value",
			input:      model.ArchivalBinaryData{Value: []byte{}},
			expectErr:  nil,
			expectData: []byte("null"),
		}, {
			name:       "with .Value being simple binary string",
			input:      model.ArchivalBinaryData{Value: []byte("Elliot")},
			expectErr:  nil,
			expectData: []byte(`{"data":"RWxsaW90","format":"base64"}`),
		}, {
			name:       "with .Value being a long binary string",
			input:      model.ArchivalBinaryData{Value: archivalBinaryInput},
			expectErr:  nil,
			expectData: archivalEncodedBinaryInput,
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				output, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this binary data", output)
				t.Logf("converted to string: %s", string(output))

				// handle errors
				switch {
				case err == nil && tc.expectErr != nil:
					t.Fatal("expected", tc.expectErr, "got", err)

				case err != nil && tc.expectErr == nil:
					t.Fatal("expected", tc.expectErr, "got", err)

				case err != nil && tc.expectErr != nil:
					if err.Error() != tc.expectErr.Error() {
						t.Fatal("expected", tc.expectErr, "got", err)
					}

				case err == nil && tc.expectErr == nil:
					// all good--fallthrough
				}

				if diff := cmp.Diff(tc.expectData, output); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test verifies that we correctly parse binary data to JSON by
	// readeding from null | {"format":"base64","data":"<base64>"}
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData model.ArchivalBinaryData
		}

		cases := []testcase{{
			name:       "with nil input array",
			input:      nil,
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalBinaryData{Value: nil},
		}, {
			name:       "with zero-length input array",
			input:      []byte{},
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalBinaryData{Value: nil},
		}, {
			name:       "with binary input that is not a complete JSON",
			input:      []byte("{"),
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalBinaryData{Value: nil},
		}, {
			name:       "with ~random binary data as input",
			input:      archivalBinaryInput,
			expectErr:  errors.New("invalid character 'W' looking for beginning of value"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with valid JSON of the wrong type (array)",
			input:      []byte("[]"),
			expectErr:  errors.New("json: cannot unmarshal array into Go value of type model.archivalBinaryDataRepr"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with valid JSON of the wrong type (number)",
			input:      []byte("1.17"),
			expectErr:  errors.New("json: cannot unmarshal number into Go value of type model.archivalBinaryDataRepr"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with input being the liternal null",
			input:      []byte(`null`),
			expectErr:  nil,
			expectData: model.ArchivalBinaryData{Value: nil},
		}, {
			name:       "with empty JSON object",
			input:      []byte("{}"),
			expectErr:  errors.New("model: invalid binary data format: ''"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with correct data model but invalid format",
			input:      []byte(`{"data":"","format":"antani"}`),
			expectErr:  errors.New("model: invalid binary data format: 'antani'"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with correct data model and format but invalid base64 string",
			input:      []byte(`{"data":"x","format":"base64"}`),
			expectErr:  errors.New("illegal base64 data at input byte 0"),
			expectData: model.ArchivalBinaryData{},
		}, {
			name:       "with correct data model and format but empty base64 string",
			input:      []byte(`{"data":"","format":"base64"}`),
			expectErr:  nil,
			expectData: model.ArchivalBinaryData{Value: []byte{}},
		}, {
			name:       "with the encoding of a simple binary string",
			input:      []byte(`{"data":"RWxsaW90","format":"base64"}`),
			expectErr:  nil,
			expectData: model.ArchivalBinaryData{Value: []byte("Elliot")},
		}, {
			name:       "with the encoding of a complex binary string",
			input:      archivalEncodedBinaryInput,
			expectErr:  nil,
			expectData: model.ArchivalBinaryData{Value: archivalBinaryInput},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// unmarshal the raw input into an ArchivalBinaryData type
				var abd model.ArchivalBinaryData
				err := json.Unmarshal(tc.input, &abd)

				t.Log("got this error", err)
				t.Log("got this .Value filed", abd.Value)
				t.Logf("converted to string: %s", string(abd.Value))

				// handle errors
				switch {
				case err == nil && tc.expectErr != nil:
					t.Fatal("expected", tc.expectErr, "got", err)

				case err != nil && tc.expectErr == nil:
					t.Fatal("expected", tc.expectErr, "got", err)

				case err != nil && tc.expectErr != nil:
					if err.Error() != tc.expectErr.Error() {
						t.Fatal("expected", tc.expectErr, "got", err)
					}

				case err == nil && tc.expectErr == nil:
					// all good--fallthrough
				}

				if diff := cmp.Diff(tc.expectData, abd); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test verifies that we correctly round trip through JSON
	t.Run("MarshalJSON then UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input model.ArchivalBinaryData
		}

		cases := []testcase{{
			name:  "with nil .Value",
			input: model.ArchivalBinaryData{Value: nil},
		}, {
			name:  "with zero length .Value",
			input: model.ArchivalBinaryData{Value: []byte{}},
		}, {
			name:  "with .Value being simple binary string",
			input: model.ArchivalBinaryData{Value: []byte("Elliot")},
		}, {
			name:  "with .Value being a long binary string",
			input: model.ArchivalBinaryData{Value: archivalBinaryInput},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				output, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this binary data", output)
				t.Logf("converted to string: %s", string(output))

				if err != nil {
					t.Fatal(err)
				}

				// parse from JSON
				var abc model.ArchivalBinaryData
				if err := json.Unmarshal(output, &abc); err != nil {
					t.Fatal(err)
				}

				// make sure we round tripped
				//
				// Note: the round trip is not perfect because the zero length value,
				// which originally is []byte{}, unmarshals to a nil value.
				//
				// Because the two are ~equivalent in Go most intents and purposes
				// and the wire representation does not change, this is OK(TM)
				diffOptions := []cmp.Option{cmpopts.EquateEmpty()}
				if diff := cmp.Diff(tc.input, abc, diffOptions...); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

func TestMaybeBinaryValue(t *testing.T) {
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
				hb := model.ArchivalMaybeBinaryData{
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
				hb := &model.ArchivalMaybeBinaryData{}
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

func TestHTTPHeader(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		tests := []struct {
			name    string                   // test name
			input   model.ArchivalHTTPHeader // what to marshal
			want    []byte                   // expected data
			wantErr bool                     // whether we expect an error
		}{{
			name: "with string value",
			input: model.ArchivalHTTPHeader{
				Key: "Content-Type",
				Value: model.ArchivalMaybeBinaryData{
					Value: "text/plain",
				},
			},
			want:    []byte(`["Content-Type","text/plain"]`),
			wantErr: false,
		}, {
			name: "with binary value",
			input: model.ArchivalHTTPHeader{
				Key: "Content-Type",
				Value: model.ArchivalMaybeBinaryData{
					Value: string(archivalBinaryInput),
				},
			},
			want:    []byte(`["Content-Type",` + string(archivalEncodedBinaryInput) + `]`),
			wantErr: false,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := tt.input.MarshalJSON()
				if (err != nil) != tt.wantErr {
					t.Fatalf("ArchivalHTTPHeader.MarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(tt.want, got); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		tests := []struct {
			name    string                   // test name
			input   []byte                   // input for the test
			want    model.ArchivalHTTPHeader // expected output
			wantErr bool                     // whether we want an error
		}{{
			name:  "with invalid input",
			input: []byte(`{}`),
			want: model.ArchivalHTTPHeader{
				Key:   "",
				Value: model.ArchivalMaybeBinaryData{Value: ""},
			},
			wantErr: true,
		}, {
			name:  "with unexpected number of items",
			input: []byte(`[]`),
			want: model.ArchivalHTTPHeader{
				Key:   "",
				Value: model.ArchivalMaybeBinaryData{Value: ""},
			},
			wantErr: true,
		}, {
			name:  "with first item not being a string",
			input: []byte(`[0,0]`),
			want: model.ArchivalHTTPHeader{
				Key:   "",
				Value: model.ArchivalMaybeBinaryData{Value: ""},
			},
			wantErr: true,
		}, {
			name:  "with both items being a string",
			input: []byte(`["x","y"]`),
			want: model.ArchivalHTTPHeader{
				Key: "x",
				Value: model.ArchivalMaybeBinaryData{
					Value: "y",
				},
			},
			wantErr: false,
		}, {
			name:  "with second item not being a map[string]interface{}",
			input: []byte(`["x",[]]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with missing format key in second item",
			input: []byte(`["x",{}]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with format value not being base64",
			input: []byte(`["x",{"format":1}]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with missing data field",
			input: []byte(`["x",{"format":"base64"}]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with data not being a string",
			input: []byte(`["x",{"format":"base64","data":1}]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with data not being base64",
			input: []byte(`["x",{"format":"base64","data":"xx"}]`),
			want: model.ArchivalHTTPHeader{
				Key: "",
				Value: model.ArchivalMaybeBinaryData{
					Value: "",
				},
			},
			wantErr: true,
		}, {
			name:  "with correctly encoded base64 data",
			input: []byte(`["x",` + string(archivalEncodedBinaryInput) + `]`),
			want: model.ArchivalHTTPHeader{
				Key: "x",
				Value: model.ArchivalMaybeBinaryData{
					Value: string(archivalBinaryInput),
				},
			},
			wantErr: false,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				hh := &model.ArchivalHTTPHeader{}
				if err := hh.UnmarshalJSON(tt.input); (err != nil) != tt.wantErr {
					t.Fatalf("ArchivalHTTPHeader.UnmarshalJSON() error = %v, wantErr %v", err, tt.wantErr)
				}
				if diff := cmp.Diff(&tt.want, hh); diff != "" {
					t.Error(diff)
				}
			})
		}
	})
}

func TestHTTPBody(t *testing.T) {
	// Implementation note: the content is always going to be the same
	// even if we modify the implementation to become:
	//
	//     type ArchivalHTTPBody ArchivalMaybeBinaryData
	//
	// instead of the correct:
	//
	//     type ArchivalHTTPBody = ArchivalMaybeBinaryData
	//
	// However, cmp.Diff also takes into account the data type. Hence, if
	// we make a mistake and apply the above change (which will in turn
	// break correct JSON serialization), the this test will fail.
	var body model.ArchivalHTTPBody
	ff := &testingx.FakeFiller{}
	ff.Fill(&body)
	data := model.ArchivalMaybeBinaryData(body)
	if diff := cmp.Diff(body, data); diff != "" {
		t.Fatal(diff)
	}
}
