package model_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
			name:       "with nil value",
			input:      nil,
			expectErr:  nil,
			expectData: []byte("null"),
		}, {
			name:       "with zero length value",
			input:      []byte{},
			expectErr:  nil,
			expectData: []byte("null"),
		}, {
			name:       "with value being a simple binary string",
			input:      []byte("Elliot"),
			expectErr:  nil,
			expectData: []byte(`{"data":"RWxsaW90","format":"base64"}`),
		}, {
			name:       "with value being a long binary string",
			input:      archivalBinaryInput,
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
	// reading from null | {"format":"base64","data":"<base64>"}
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
			expectData: nil,
		}, {
			name:       "with zero-length input array",
			input:      []byte{},
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: nil,
		}, {
			name:       "with binary input that is not a complete JSON",
			input:      []byte("{"),
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: nil,
		}, {
			name:       "with ~random binary data as input",
			input:      archivalBinaryInput,
			expectErr:  errors.New("invalid character 'W' looking for beginning of value"),
			expectData: nil,
		}, {
			name:       "with valid JSON of the wrong type (array)",
			input:      []byte("[]"),
			expectErr:  errors.New("json: cannot unmarshal array into Go value of type model.archivalBinaryDataRepr"),
			expectData: nil,
		}, {
			name:       "with valid JSON of the wrong type (number)",
			input:      []byte("1.17"),
			expectErr:  errors.New("json: cannot unmarshal number into Go value of type model.archivalBinaryDataRepr"),
			expectData: nil,
		}, {
			name:       "with input being the liternal null",
			input:      []byte(`null`),
			expectErr:  nil,
			expectData: nil,
		}, {
			name:       "with empty JSON object",
			input:      []byte("{}"),
			expectErr:  errors.New("model: invalid binary data format: ''"),
			expectData: nil,
		}, {
			name:       "with correct data model but invalid format",
			input:      []byte(`{"data":"","format":"antani"}`),
			expectErr:  errors.New("model: invalid binary data format: 'antani'"),
			expectData: nil,
		}, {
			name:       "with correct data model and format but invalid base64 string",
			input:      []byte(`{"data":"x","format":"base64"}`),
			expectErr:  errors.New("illegal base64 data at input byte 0"),
			expectData: nil,
		}, {
			name:       "with correct data model and format but empty base64 string",
			input:      []byte(`{"data":"","format":"base64"}`),
			expectErr:  nil,
			expectData: []byte{},
		}, {
			name:       "with the encoding of a simple binary string",
			input:      []byte(`{"data":"RWxsaW90","format":"base64"}`),
			expectErr:  nil,
			expectData: []byte("Elliot"),
		}, {
			name:       "with the encoding of a complex binary string",
			input:      archivalEncodedBinaryInput,
			expectErr:  nil,
			expectData: archivalBinaryInput,
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// unmarshal the raw input into an ArchivalBinaryData type
				var abd model.ArchivalBinaryData
				err := json.Unmarshal(tc.input, &abd)

				t.Log("got this error", err)
				t.Log("got this []byte-like value", abd)
				t.Logf("converted to string: %s", string(abd))

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
			name:  "with nil value",
			input: nil,
		}, {
			name:  "with zero length value",
			input: []byte{},
		}, {
			name:  "with value being a simple binary string",
			input: []byte("Elliot"),
		}, {
			name:  "with value being a long binary string",
			input: archivalBinaryInput,
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

func TestArchivalScrubbedMaybeBinaryString(t *testing.T) {
	t.Run("Supports assignment from a nil byte array", func(t *testing.T) {
		var data []byte = nil // explicit
		casted := model.ArchivalScrubbedMaybeBinaryString(data)
		if casted != "" {
			t.Fatal("unexpected value")
		}
	})

	// This test verifies that we correctly serialize a string to JSON by
	// producing "" | {"format":"base64","data":"<base64>"}
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the possibly-binary input
			input model.ArchivalScrubbedMaybeBinaryString

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name:       "with empty string value",
			input:      "",
			expectErr:  nil,
			expectData: []byte(`""`),
		}, {
			name:       "with value being a textual string",
			input:      "Elliot",
			expectErr:  nil,
			expectData: []byte(`"Elliot"`),
		}, {
			name:       "with value being a long binary string",
			input:      model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput),
			expectErr:  nil,
			expectData: archivalEncodedBinaryInput,
		}, {
			name:       "with string containing IP addresses and endpoints",
			input:      "a 130.192.91.211 b ::1 c [::1]:443 d 130.192.91.211:80",
			expectErr:  nil,
			expectData: []byte(`"a [scrubbed] b [scrubbed] c [scrubbed] d [scrubbed]"`),
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
	// reading from "" | {"format":"base64","data":"<base64>"}
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect
			expectData model.ArchivalScrubbedMaybeBinaryString
		}

		cases := []testcase{{
			name:       "with nil input array",
			input:      nil,
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with zero-length input array",
			input:      []byte{},
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with binary input that is not a complete JSON",
			input:      []byte("{"),
			expectErr:  errors.New("unexpected end of JSON input"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with ~random binary data as input",
			input:      archivalBinaryInput,
			expectErr:  errors.New("invalid character 'W' looking for beginning of value"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with valid JSON of the wrong type (array)",
			input:      []byte("[]"),
			expectErr:  errors.New("json: cannot unmarshal array into Go value of type model.archivalBinaryDataRepr"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with valid JSON of the wrong type (number)",
			input:      []byte("1.17"),
			expectErr:  errors.New("json: cannot unmarshal number into Go value of type model.archivalBinaryDataRepr"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with input being the liternal null",
			input:      []byte(`null`),
			expectErr:  nil,
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with empty JSON object",
			input:      []byte("{}"),
			expectErr:  errors.New("model: invalid binary data format: ''"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with correct data model but invalid format",
			input:      []byte(`{"data":"","format":"antani"}`),
			expectErr:  errors.New("model: invalid binary data format: 'antani'"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with correct data model and format but invalid base64 string",
			input:      []byte(`{"data":"x","format":"base64"}`),
			expectErr:  errors.New("illegal base64 data at input byte 0"),
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with correct data model and format but empty base64 string",
			input:      []byte(`{"data":"","format":"base64"}`),
			expectErr:  nil,
			expectData: model.ArchivalScrubbedMaybeBinaryString(""),
		}, {
			name:       "with the a string",
			input:      []byte(`"Elliot"`),
			expectErr:  nil,
			expectData: model.ArchivalScrubbedMaybeBinaryString("Elliot"),
		}, {
			name:       "with the encoding of a string",
			input:      []byte(`{"data":"RWxsaW90","format":"base64"}`),
			expectErr:  nil,
			expectData: model.ArchivalScrubbedMaybeBinaryString("Elliot"),
		}, {
			name:       "with the encoding of a complex binary string",
			input:      archivalEncodedBinaryInput,
			expectErr:  nil,
			expectData: model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// unmarshal the raw input into an ArchivalBinaryData type
				var abd model.ArchivalScrubbedMaybeBinaryString
				err := json.Unmarshal(tc.input, &abd)

				t.Log("got this error", err)
				t.Log("got this maybe-binary-string value", abd)
				t.Logf("converted to string: %s", string(abd))

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

			// input is the maybe-binary input
			input model.ArchivalScrubbedMaybeBinaryString
		}

		cases := []testcase{{
			name:  "with empty value",
			input: "",
		}, {
			name:  "with value being a simple textual string",
			input: "Elliot",
		}, {
			name:  "with value being a long binary string",
			input: model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput),
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
				var abc model.ArchivalScrubbedMaybeBinaryString
				if err := json.Unmarshal(output, &abc); err != nil {
					t.Fatal(err)
				}

				// make sure we round tripped
				if diff := cmp.Diff(tc.input, abc); diff != "" {
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

func TestArchivalHTTPHeader(t *testing.T) {
	t.Run("MarshalJSON", func(t *testing.T) {
		tests := []struct {
			name    string                   // test name
			input   model.ArchivalHTTPHeader // what to marshal
			want    []byte                   // expected data
			wantErr bool                     // whether we expect an error
		}{{
			name: "with string value",
			input: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString("Content-Type"),
				model.ArchivalScrubbedMaybeBinaryString("text/plain"),
			},
			want:    []byte(`["Content-Type","text/plain"]`),
			wantErr: false,
		}, {
			name: "with binary value",
			input: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString("Content-Type"),
				model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput),
			},
			want:    []byte(`["Content-Type",` + string(archivalEncodedBinaryInput) + `]`),
			wantErr: false,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				got, err := json.Marshal(tt.input)
				if (err != nil) != tt.wantErr {
					t.Fatalf("json.Marshal() error = %v, wantErr %v", err, tt.wantErr)
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
			wantErr error                    // whether we want an error
		}{{
			name:  "with invalid input",
			input: []byte(`{}`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("json: cannot unmarshal object into Go value of type []model.ArchivalScrubbedMaybeBinaryString"),
		}, {
			name:  "with zero items",
			input: []byte(`[]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("invalid ArchivalHTTPHeader: expected 2 elements, got 0"),
		}, {
			name:    "with just one item",
			input:   []byte(`["x"]`),
			want:    [2]model.ArchivalScrubbedMaybeBinaryString{},
			wantErr: errors.New("invalid ArchivalHTTPHeader: expected 2 elements, got 1"),
		}, {
			name:    "with three items",
			input:   []byte(`["x","x","x"]`),
			want:    [2]model.ArchivalScrubbedMaybeBinaryString{},
			wantErr: errors.New("invalid ArchivalHTTPHeader: expected 2 elements, got 3"),
		}, {
			name:  "with first item not being a string",
			input: []byte(`[0,0]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("json: cannot unmarshal number into Go value of type model.archivalBinaryDataRepr"),
		}, {
			name:  "with both items being a string",
			input: []byte(`["x","y"]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString("x"),
				model.ArchivalScrubbedMaybeBinaryString("y"),
			},
			wantErr: nil,
		}, {
			name:  "with second item not being a map[string]interface{}",
			input: []byte(`["x",[]]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("json: cannot unmarshal array into Go value of type model.archivalBinaryDataRepr"),
		}, {
			name:  "with missing format key in second item",
			input: []byte(`["x",{}]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("model: invalid binary data format: ''"),
		}, {
			name:  "with format value not being base64",
			input: []byte(`["x",{"format":1}]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("json: cannot unmarshal number into Go struct field archivalBinaryDataRepr.format of type string"),
		}, {
			name:  "with missing data field",
			input: []byte(`["x",{"format":"base64"}]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString("x"),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: nil,
		}, {
			name:  "with data not being a string",
			input: []byte(`["x",{"format":"base64","data":1}]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("json: cannot unmarshal number into Go struct field archivalBinaryDataRepr.data of type []uint8"),
		}, {
			name:  "with data not being base64",
			input: []byte(`["x",{"format":"base64","data":"xx"}]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString(""),
				model.ArchivalScrubbedMaybeBinaryString(""),
			},
			wantErr: errors.New("illegal base64 data at input byte 0"),
		}, {
			name:  "with correctly encoded base64 data",
			input: []byte(`["x",` + string(archivalEncodedBinaryInput) + `]`),
			want: model.ArchivalHTTPHeader{
				model.ArchivalScrubbedMaybeBinaryString("x"),
				model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput),
			},
			wantErr: nil,
		}}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var hh model.ArchivalHTTPHeader

				err := json.Unmarshal(tt.input, &hh)

				switch {
				case err != nil && tt.wantErr != nil:
					if err.Error() != tt.wantErr.Error() {
						t.Fatal("expected", tt.wantErr, "got", err)
					}

				case err != nil && tt.wantErr == nil:
					t.Fatal("expected", tt.wantErr, "got", err)
				case err == nil && tt.wantErr != nil:
					t.Fatal("expected", tt.wantErr, "got", err)

				case err == nil && tt.wantErr == nil:
					// note: only check the result when there is no error
					if diff := cmp.Diff(tt.want, hh); diff != "" {
						t.Error(diff)
					}
				}
			})
		}
	})
}

// This test ensures that ArchivalDNSLookupResult is WAI
func TestArchivalDNSLookupResult(t *testing.T) {

	// This test ensures that we correctly serialize to JSON.
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the input struct
			input model.ArchivalDNSLookupResult

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name: "serialization of a successful DNS lookup",
			input: model.ArchivalDNSLookupResult{
				Answers: []model.ArchivalDNSAnswer{{
					ASN:        15169,
					ASOrgName:  "Google LLC",
					AnswerType: "A",
					Hostname:   "",
					IPv4:       "8.8.8.8",
					IPv6:       "",
					TTL:        nil,
				}, {
					ASN:        15169,
					ASOrgName:  "Google LLC",
					AnswerType: "AAAA",
					Hostname:   "",
					IPv4:       "",
					IPv6:       "2001:4860:4860::8888",
					TTL:        nil,
				}},
				Engine:           "getaddrinfo",
				Failure:          nil,
				GetaddrinfoError: 0,
				Hostname:         "dns.google",
				QueryType:        "ANY",
				RawResponse:      nil,
				Rcode:            0,
				ResolverHostname: nil,
				ResolverPort:     nil,
				ResolverAddress:  "",
				T0:               0.5,
				T:                0.7,
				Tags:             []string{"dns"},
				TransactionID:    44,
			},
			expectErr:  nil,
			expectData: []byte(`{"answers":[{"asn":15169,"as_org_name":"Google LLC","answer_type":"A","ipv4":"8.8.8.8","ttl":null},{"asn":15169,"as_org_name":"Google LLC","answer_type":"AAAA","ipv6":"2001:4860:4860::8888","ttl":null}],"engine":"getaddrinfo","failure":null,"hostname":"dns.google","query_type":"ANY","resolver_hostname":null,"resolver_port":null,"resolver_address":"","t0":0.5,"t":0.7,"tags":["dns"],"transaction_id":44}`),
		}, {
			name: "serialization of a failed DNS lookup",
			input: model.ArchivalDNSLookupResult{
				Answers: nil,
				Engine:  "getaddrinfo",
				Failure: (func() *string {
					s := netxlite.FailureDNSNXDOMAINError
					return &s
				}()),
				GetaddrinfoError: 5,
				Hostname:         "dns.googlex",
				QueryType:        "ANY",
				RawResponse:      nil,
				Rcode:            0,
				ResolverHostname: nil,
				ResolverPort:     nil,
				ResolverAddress:  "",
				T0:               0.5,
				T:                0.77,
				Tags:             []string{"dns"},
				TransactionID:    43,
			},
			expectErr:  nil,
			expectData: []byte(`{"answers":null,"engine":"getaddrinfo","failure":"dns_nxdomain_error","getaddrinfo_error":5,"hostname":"dns.googlex","query_type":"ANY","resolver_hostname":null,"resolver_port":null,"resolver_address":"","t0":0.5,"t":0.77,"tags":["dns"],"transaction_id":43}`),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				data, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this raw data", data)
				t.Logf("converted to string: %s", string(data))

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

				// make sure the serialization is OK
				if diff := cmp.Diff(tc.expectData, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test ensures that we can unmarshal from the JSON representation
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectStruct is the struct we expect to see
			expectStruct model.ArchivalDNSLookupResult
		}

		cases := []testcase{{
			name:      "deserialization of a successful DNS lookup",
			expectErr: nil,
			input:     []byte(`{"answers":[{"asn":15169,"as_org_name":"Google LLC","answer_type":"A","ipv4":"8.8.8.8","ttl":null},{"asn":15169,"as_org_name":"Google LLC","answer_type":"AAAA","ipv6":"2001:4860:4860::8888","ttl":null}],"engine":"getaddrinfo","failure":null,"hostname":"dns.google","query_type":"ANY","resolver_hostname":null,"resolver_port":null,"resolver_address":"","t0":0.5,"t":0.7,"tags":["dns"],"transaction_id":44}`),
			expectStruct: model.ArchivalDNSLookupResult{
				Answers: []model.ArchivalDNSAnswer{{
					ASN:        15169,
					ASOrgName:  "Google LLC",
					AnswerType: "A",
					Hostname:   "",
					IPv4:       "8.8.8.8",
					IPv6:       "",
					TTL:        nil,
				}, {
					ASN:        15169,
					ASOrgName:  "Google LLC",
					AnswerType: "AAAA",
					Hostname:   "",
					IPv4:       "",
					IPv6:       "2001:4860:4860::8888",
					TTL:        nil,
				}},
				Engine:           "getaddrinfo",
				Failure:          nil,
				GetaddrinfoError: 0,
				Hostname:         "dns.google",
				QueryType:        "ANY",
				RawResponse:      nil,
				Rcode:            0,
				ResolverHostname: nil,
				ResolverPort:     nil,
				ResolverAddress:  "",
				T0:               0.5,
				T:                0.7,
				Tags:             []string{"dns"},
				TransactionID:    44,
			},
		}, {
			name:      "deserialization of a failed DNS lookup",
			input:     []byte(`{"answers":null,"engine":"getaddrinfo","failure":"dns_nxdomain_error","getaddrinfo_error":5,"hostname":"dns.googlex","query_type":"ANY","resolver_hostname":null,"resolver_port":null,"resolver_address":"","t0":0.5,"t":0.77,"tags":["dns"],"transaction_id":43}`),
			expectErr: nil,
			expectStruct: model.ArchivalDNSLookupResult{
				Answers: nil,
				Engine:  "getaddrinfo",
				Failure: (func() *string {
					s := netxlite.FailureDNSNXDOMAINError
					return &s
				}()),
				GetaddrinfoError: 5,
				Hostname:         "dns.googlex",
				QueryType:        "ANY",
				RawResponse:      nil,
				Rcode:            0,
				ResolverHostname: nil,
				ResolverPort:     nil,
				ResolverAddress:  "",
				T0:               0.5,
				T:                0.77,
				Tags:             []string{"dns"},
				TransactionID:    43,
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// parse the JSON
				var data model.ArchivalDNSLookupResult
				err := json.Unmarshal(tc.input, &data)

				t.Log("got this error", err)
				t.Logf("got this struct %+v", data)

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

				// make sure the deserialization is OK
				if diff := cmp.Diff(tc.expectStruct, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

// This test ensures that ArchivalTCPConnectResult is WAI
func TestArchivalTCPConnectResult(t *testing.T) {

	// This test ensures that we correctly serialize to JSON.
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the input struct
			input model.ArchivalTCPConnectResult

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name: "serialization of a successful TCP connect",
			input: model.ArchivalTCPConnectResult{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: nil,
					Failure: nil,
					Success: true,
				},
				T0:            4,
				T:             7,
				Tags:          []string{"tcp"},
				TransactionID: 99,
			},
			expectErr:  nil,
			expectData: []byte(`{"ip":"8.8.8.8","port":443,"status":{"failure":null,"success":true},"t0":4,"t":7,"tags":["tcp"],"transaction_id":99}`),
		}, {
			name: "serialization of a failed TCP connect",
			input: model.ArchivalTCPConnectResult{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: nil,
					Failure: (func() *string {
						s := netxlite.FailureGenericTimeoutError
						return &s
					}()),
					Success: false,
				},
				T0:            4,
				T:             7,
				Tags:          []string{"tcp"},
				TransactionID: 99,
			},
			expectErr:  nil,
			expectData: []byte(`{"ip":"8.8.8.8","port":443,"status":{"failure":"generic_timeout_error","success":false},"t0":4,"t":7,"tags":["tcp"],"transaction_id":99}`),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				data, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this raw data", data)
				t.Logf("converted to string: %s", string(data))

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

				// make sure the serialization is OK
				if diff := cmp.Diff(tc.expectData, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test ensures that we can unmarshal from the JSON representation
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectStruct is the struct we expect to see
			expectStruct model.ArchivalTCPConnectResult
		}

		cases := []testcase{{
			name:      "deserialization of a successful TCP connect",
			expectErr: nil,
			input:     []byte(`{"ip":"8.8.8.8","port":443,"status":{"failure":null,"success":true},"t0":4,"t":7,"tags":["tcp"],"transaction_id":99}`),
			expectStruct: model.ArchivalTCPConnectResult{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: nil,
					Failure: nil,
					Success: true,
				},
				T0:            4,
				T:             7,
				Tags:          []string{"tcp"},
				TransactionID: 99,
			},
		}, {
			name:      "deserialization of a failed TCP connect",
			input:     []byte(`{"ip":"8.8.8.8","port":443,"status":{"failure":"generic_timeout_error","success":false},"t0":4,"t":7,"tags":["tcp"],"transaction_id":99}`),
			expectErr: nil,
			expectStruct: model.ArchivalTCPConnectResult{
				IP:   "8.8.8.8",
				Port: 443,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: nil,
					Failure: (func() *string {
						s := netxlite.FailureGenericTimeoutError
						return &s
					}()),
					Success: false,
				},
				T0:            4,
				T:             7,
				Tags:          []string{"tcp"},
				TransactionID: 99,
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// parse the JSON
				var data model.ArchivalTCPConnectResult
				err := json.Unmarshal(tc.input, &data)

				t.Log("got this error", err)
				t.Logf("got this struct %+v", data)

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

				// make sure the deserialization is OK
				if diff := cmp.Diff(tc.expectStruct, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

// This test ensures that ArchivalTLSOrQUICHandshakeResult is WAI
func TestArchivalTLSOrQUICHandshakeResult(t *testing.T) {

	// This test ensures that we correctly serialize to JSON.
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the input struct
			input model.ArchivalTLSOrQUICHandshakeResult

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name: "serialization of a successful TLS handshake",
			input: model.ArchivalTLSOrQUICHandshakeResult{
				Network:            "tcp",
				Address:            "8.8.8.8:443",
				CipherSuite:        "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
				Failure:            nil,
				SoError:            nil,
				NegotiatedProtocol: "http/1.1",
				NoTLSVerify:        false,
				PeerCertificates: []model.ArchivalBinaryData{
					model.ArchivalBinaryData(archivalBinaryInput),
				},
				ServerName:    "dns.google",
				T0:            1.0,
				T:             2.0,
				Tags:          []string{"tls"},
				TLSVersion:    "TLSv1.3",
				TransactionID: 14,
			},
			expectErr:  nil,
			expectData: []byte(`{"network":"tcp","address":"8.8.8.8:443","cipher_suite":"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256","failure":null,"negotiated_protocol":"http/1.1","no_tls_verify":false,"peer_certificates":[{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}],"server_name":"dns.google","t0":1,"t":2,"tags":["tls"],"tls_version":"TLSv1.3","transaction_id":14}`),
		}, {
			name: "serialization of a failed TLS handshake",
			input: model.ArchivalTLSOrQUICHandshakeResult{
				Network:     "tcp",
				Address:     "8.8.8.8:443",
				CipherSuite: "",
				Failure: (func() *string {
					s := netxlite.FailureConnectionReset
					return &s
				})(),
				SoError: (func() *string {
					s := "connection reset by peer"
					return &s
				})(),
				NegotiatedProtocol: "",
				NoTLSVerify:        false,
				PeerCertificates:   []model.ArchivalBinaryData{},
				ServerName:         "dns.google",
				T0:                 1.0,
				T:                  2.0,
				Tags:               []string{"tls"},
				TLSVersion:         "",
				TransactionID:      4,
			},
			expectErr:  nil,
			expectData: []byte(`{"network":"tcp","address":"8.8.8.8:443","cipher_suite":"","failure":"connection_reset","so_error":"connection reset by peer","negotiated_protocol":"","no_tls_verify":false,"peer_certificates":[],"server_name":"dns.google","t0":1,"t":2,"tags":["tls"],"tls_version":"","transaction_id":4}`),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				data, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this raw data", data)
				t.Logf("converted to string: %s", string(data))

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

				// make sure the serialization is OK
				if diff := cmp.Diff(tc.expectData, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test ensures that we can unmarshal from the JSON representation
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectStruct is the struct we expect to see
			expectStruct model.ArchivalTLSOrQUICHandshakeResult
		}

		cases := []testcase{{
			name:      "deserialization of a successful TLS handshake",
			expectErr: nil,
			input:     []byte(`{"network":"tcp","address":"8.8.8.8:443","cipher_suite":"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256","failure":null,"negotiated_protocol":"http/1.1","no_tls_verify":false,"peer_certificates":[{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6xNyawT2yK+en3ZJKJSadiJwdFXqgQrotixGfbVETm7gM+G+V+djKv1xXQkOqLUQE7XEB8=","format":"base64"}],"server_name":"dns.google","t0":1,"t":2,"tags":["tls"],"tls_version":"TLSv1.3","transaction_id":14}`),
			expectStruct: model.ArchivalTLSOrQUICHandshakeResult{
				Network:            "tcp",
				Address:            "8.8.8.8:443",
				CipherSuite:        "TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
				Failure:            nil,
				SoError:            nil,
				NegotiatedProtocol: "http/1.1",
				NoTLSVerify:        false,
				PeerCertificates: []model.ArchivalBinaryData{
					model.ArchivalBinaryData(archivalBinaryInput),
				},
				ServerName:    "dns.google",
				T0:            1.0,
				T:             2.0,
				Tags:          []string{"tls"},
				TLSVersion:    "TLSv1.3",
				TransactionID: 14,
			},
		}, {
			name:      "deserialization of a failed TLS handshake",
			input:     []byte(`{"network":"tcp","address":"8.8.8.8:443","cipher_suite":"","failure":"connection_reset","so_error":"connection reset by peer","negotiated_protocol":"","no_tls_verify":false,"peer_certificates":[],"server_name":"dns.google","t0":1,"t":2,"tags":["tls"],"tls_version":"","transaction_id":4}`),
			expectErr: nil,
			expectStruct: model.ArchivalTLSOrQUICHandshakeResult{
				Network:     "tcp",
				Address:     "8.8.8.8:443",
				CipherSuite: "",
				Failure: (func() *string {
					s := netxlite.FailureConnectionReset
					return &s
				})(),
				SoError: (func() *string {
					s := "connection reset by peer"
					return &s
				})(),
				NegotiatedProtocol: "",
				NoTLSVerify:        false,
				PeerCertificates:   []model.ArchivalBinaryData{},
				ServerName:         "dns.google",
				T0:                 1.0,
				T:                  2.0,
				Tags:               []string{"tls"},
				TLSVersion:         "",
				TransactionID:      4,
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// parse the JSON
				var data model.ArchivalTLSOrQUICHandshakeResult
				err := json.Unmarshal(tc.input, &data)

				t.Log("got this error", err)
				t.Logf("got this struct %+v", data)

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

				// make sure the deserialization is OK
				if diff := cmp.Diff(tc.expectStruct, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

// This test ensures that ArchivalHTTPRequestResult is WAI
func TestArchivalHTTPRequestResult(t *testing.T) {

	// This test ensures that we correctly serialize to JSON.
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the input struct
			input model.ArchivalHTTPRequestResult

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{

			// This test ensures we can serialize a typical, successful HTTP measurement
			{
				name: "serialization of a successful HTTP request",
				input: model.ArchivalHTTPRequestResult{
					Network: "tcp",
					Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
					ALPN:    "h2",
					Failure: nil,
					Request: model.ArchivalHTTPRequest{
						Body:            model.ArchivalScrubbedMaybeBinaryString(""),
						BodyIsTruncated: false,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Accept"),
							model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
							model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
							"User-Agent": "miniooni/0.1.0",
						},
						Method: "GET",
						Tor: model.ArchivalHTTPTor{
							ExitIP:   nil,
							ExitName: nil,
							IsTor:    false,
						},
						Transport: "tcp",
						URL:       "https://www.example.com/",
					},
					Response: model.ArchivalHTTPResponse{
						Body: model.ArchivalScrubbedMaybeBinaryString(
							"Bonsoir, Elliot!",
						),
						BodyIsTruncated: false,
						Code:            200,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Age"),
							model.ArchivalScrubbedMaybeBinaryString("131833"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Server"),
							model.ArchivalScrubbedMaybeBinaryString("Apache"),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Age":    "131833",
							"Server": "Apache",
						},
						Locations: nil,
					},
					T0:            0.7,
					T:             1.33,
					Tags:          []string{"http"},
					TransactionID: 5,
				},
				expectErr:  nil,
				expectData: []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":null,"request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":"Bonsoir, Elliot!","body_is_truncated":false,"code":200,"headers_list":[["Age","131833"],["Server","Apache"]],"headers":{"Age":"131833","Server":"Apache"}},"t0":0.7,"t":1.33,"tags":["http"],"transaction_id":5}`),
			},

			// This test ensures we can serialize a typical failed HTTP measurement
			{
				name: "serialization of a failed HTTP request",
				input: model.ArchivalHTTPRequestResult{
					Network: "tcp",
					Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
					ALPN:    "h2",
					Failure: (func() *string {
						s := netxlite.FailureGenericTimeoutError
						return &s
					})(),
					Request: model.ArchivalHTTPRequest{
						Body:            model.ArchivalScrubbedMaybeBinaryString(""),
						BodyIsTruncated: false,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Accept"),
							model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
							model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
							"User-Agent": "miniooni/0.1.0",
						},
						Method: "GET",
						Tor: model.ArchivalHTTPTor{
							ExitIP:   nil,
							ExitName: nil,
							IsTor:    false,
						},
						Transport: "tcp",
						URL:       "https://www.example.com/",
					},
					Response: model.ArchivalHTTPResponse{
						Body:            model.ArchivalScrubbedMaybeBinaryString(""),
						BodyIsTruncated: false,
						Code:            0,
						HeadersList:     []model.ArchivalHTTPHeader{},
						Headers:         map[string]model.ArchivalScrubbedMaybeBinaryString{},
						Locations:       nil,
					},
					T0:            0.4,
					T:             1.563,
					Tags:          []string{"http"},
					TransactionID: 6,
				},
				expectErr:  nil,
				expectData: []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":"generic_timeout_error","request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":"","body_is_truncated":false,"code":0,"headers_list":[],"headers":{}},"t0":0.4,"t":1.563,"tags":["http"],"transaction_id":6}`),
			},

			// This test ensures we can correctly serialize an HTTP measurement where the
			// response body and some headers contain binary data
			//
			// We need this test to continue to have confidence that our serialization
			// code is always correctly handling how we generate JSONs
			{
				name: "serialization of a successful HTTP request with binary data",
				input: model.ArchivalHTTPRequestResult{
					Network: "tcp",
					Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
					ALPN:    "h2",
					Failure: nil,
					Request: model.ArchivalHTTPRequest{
						Body:            model.ArchivalScrubbedMaybeBinaryString(""),
						BodyIsTruncated: false,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Accept"),
							model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
							model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Antani"),
							model.ArchivalScrubbedMaybeBinaryString(string(archivalBinaryInput[:7])),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Antani"),
							model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput[7:14]),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
							"User-Agent": "miniooni/0.1.0",
							"Antani":     model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput[:7]),
						},
						Method: "GET",
						Tor: model.ArchivalHTTPTor{
							ExitIP:   nil,
							ExitName: nil,
							IsTor:    false,
						},
						Transport: "tcp",
						URL:       "https://www.example.com/",
					},
					Response: model.ArchivalHTTPResponse{
						Body: model.ArchivalScrubbedMaybeBinaryString(
							archivalBinaryInput[:77],
						),
						BodyIsTruncated: false,
						Code:            200,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Age"),
							model.ArchivalScrubbedMaybeBinaryString("131833"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Server"),
							model.ArchivalScrubbedMaybeBinaryString("Apache"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Mascetti"),
							model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput[14:21]),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Mascetti"),
							model.ArchivalScrubbedMaybeBinaryString(archivalBinaryInput[21:28]),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Age":      "131833",
							"Server":   "Apache",
							"Mascetti": model.ArchivalScrubbedMaybeBinaryString(archivalEncodedBinaryInput[14:21]),
						},
						Locations: nil,
					},
					T0:            0.7,
					T:             1.33,
					Tags:          []string{"http"},
					TransactionID: 5,
				},
				expectErr:  nil,
				expectData: []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":null,"request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"],["Antani",{"data":"V+V5+6a7DQ==","format":"base64"}],["Antani",{"data":"vM69p6C6pA==","format":"base64"}]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","Antani":{"data":"V+V5+6a7DQ==","format":"base64"},"User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":{"data":"V+V5+6a7DbzOvaeguqR4eBJZ7mg5pAeYxT68Vcv+NDx+G1qzIp3BLW7KW/EQJUceROItYAjqsArMBUig9Xg48Ns/nZ8lb4kAlpOvQ6w=","format":"base64"},"body_is_truncated":false,"code":200,"headers_list":[["Age","131833"],["Server","Apache"],["Mascetti",{"data":"eHgSWe5oOQ==","format":"base64"}],["Mascetti",{"data":"pAeYxT68VQ==","format":"base64"}]],"headers":{"Age":"131833","Mascetti":"6a7DbzO","Server":"Apache"}},"t0":0.7,"t":1.33,"tags":["http"],"transaction_id":5}`),
			},

			// This test ensures we can serialize an HTTP measurement containing
			// IP addresses in the headers or the body.
			//
			// This test will fail until we implement more aggressive scrubbing, which
			// is poised to happen as part of https://github.com/ooni/probe/issues/2531,
			// where we implemented happy eyeballs, which may lead to surprises, so
			// we want to be proactive and scrub more than before.
			//
			// We need this test to continue to have confidence that our serialization
			// code is always correctly handling how we generate JSONs.
			{
				name: "serialization of a successful HTTP request with IP addresses and endpoints",
				input: model.ArchivalHTTPRequestResult{
					Network: "tcp",
					Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
					ALPN:    "h2",
					Failure: nil,
					Request: model.ArchivalHTTPRequest{
						Body:            model.ArchivalScrubbedMaybeBinaryString(""),
						BodyIsTruncated: false,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Accept"),
							model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
							model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("AntaniV4"),
							model.ArchivalScrubbedMaybeBinaryString("130.192.91.211"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("AntaniV6"),
							model.ArchivalScrubbedMaybeBinaryString("2606:2800:220:1:248:1893:25c8:1946"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("AntaniV4Epnt"),
							model.ArchivalScrubbedMaybeBinaryString("130.192.91.211:443"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("AntaniV6Epnt"),
							model.ArchivalScrubbedMaybeBinaryString("[2606:2800:220:1:248:1893:25c8:1946]:5222"),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Accept":       "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
							"User-Agent":   "miniooni/0.1.0",
							"AntaniV4":     "130.192.91.211",
							"AntaniV6":     "2606:2800:220:1:248:1893:25c8:1946",
							"AntaniV4Epnt": "130.192.91.211:443",
							"AntaniV6Epnt": "[2606:2800:220:1:248:1893:25c8:1946]:5222",
						},
						Method: "GET",
						Tor: model.ArchivalHTTPTor{
							ExitIP:   nil,
							ExitName: nil,
							IsTor:    false,
						},
						Transport: "tcp",
						URL:       "https://www.example.com/",
					},
					Response: model.ArchivalHTTPResponse{
						Body: model.ArchivalScrubbedMaybeBinaryString(
							"<HTML><BODY>Your address is 130.192.91.211 and 2606:2800:220:1:248:1893:25c8:1946 and you have endpoints [2606:2800:220:1:248:1893:25c8:1946]:5222 and 130.192.91.211:443. You're welcome.</BODY></HTML>",
						),
						BodyIsTruncated: false,
						Code:            200,
						HeadersList: []model.ArchivalHTTPHeader{{
							model.ArchivalScrubbedMaybeBinaryString("Age"),
							model.ArchivalScrubbedMaybeBinaryString("131833"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("Server"),
							model.ArchivalScrubbedMaybeBinaryString("Apache"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("MascettiV4"),
							model.ArchivalScrubbedMaybeBinaryString("130.192.91.211"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("MascettiV6"),
							model.ArchivalScrubbedMaybeBinaryString("2606:2800:220:1:248:1893:25c8:1946"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("MascettiV4Epnt"),
							model.ArchivalScrubbedMaybeBinaryString("130.192.91.211:443"),
						}, {
							model.ArchivalScrubbedMaybeBinaryString("MascettiV6Epnt"),
							model.ArchivalScrubbedMaybeBinaryString("[2606:2800:220:1:248:1893:25c8:1946]:5222"),
						}},
						Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
							"Age":            "131833",
							"Server":         "Apache",
							"MascettiV4":     "130.192.91.211",
							"MascettiV6":     "2606:2800:220:1:248:1893:25c8:1946",
							"MascettiV4Epnt": "130.192.91.211:443",
							"MascettiV6Epnt": "[2606:2800:220:1:248:1893:25c8:1946]:5222",
						},
						Locations: nil,
					},
					T0:            0.7,
					T:             1.33,
					Tags:          []string{"http"},
					TransactionID: 5,
				},
				expectErr:  nil,
				expectData: []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":null,"request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"],["AntaniV4","[scrubbed]"],["AntaniV6","[scrubbed]"],["AntaniV4Epnt","[scrubbed]"],["AntaniV6Epnt","[scrubbed]"]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","AntaniV4":"[scrubbed]","AntaniV4Epnt":"[scrubbed]","AntaniV6":"[scrubbed]","AntaniV6Epnt":"[scrubbed]","User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":"\u003cHTML\u003e\u003cBODY\u003eYour address is [scrubbed] and [scrubbed] and you have endpoints [scrubbed] and [scrubbed]. You're welcome.\u003c/BODY\u003e\u003c/HTML\u003e","body_is_truncated":false,"code":200,"headers_list":[["Age","131833"],["Server","Apache"],["MascettiV4","[scrubbed]"],["MascettiV6","[scrubbed]"],["MascettiV4Epnt","[scrubbed]"],["MascettiV6Epnt","[scrubbed]"]],"headers":{"Age":"131833","MascettiV4":"[scrubbed]","MascettiV4Epnt":"[scrubbed]","MascettiV6":"[scrubbed]","MascettiV6Epnt":"[scrubbed]","Server":"Apache"}},"t0":0.7,"t":1.33,"tags":["http"],"transaction_id":5}`),
			},
		}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				data, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this raw data", data)
				t.Logf("converted to string: %s", string(data))

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

				// make sure the serialization is OK
				if diff := cmp.Diff(tc.expectData, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test ensures that we can unmarshal from the JSON representation
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectStruct is the struct we expect to see
			expectStruct model.ArchivalHTTPRequestResult
		}

		cases := []testcase{{
			name:      "deserialization of a successful HTTP request",
			expectErr: nil,
			input:     []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":null,"request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":"Bonsoir, Elliot!","body_is_truncated":false,"code":200,"headers_list":[["Age","131833"],["Server","Apache"]],"headers":{"Age":"131833","Server":"Apache"}},"t0":0.7,"t":1.33,"tags":["http"],"transaction_id":5}`),
			expectStruct: model.ArchivalHTTPRequestResult{
				Network: "tcp",
				Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
				ALPN:    "h2",
				Failure: nil,
				Request: model.ArchivalHTTPRequest{
					Body:            model.ArchivalScrubbedMaybeBinaryString(""),
					BodyIsTruncated: false,
					HeadersList: []model.ArchivalHTTPHeader{{
						model.ArchivalScrubbedMaybeBinaryString("Accept"),
						model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
					}, {
						model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
						model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
					}},
					Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
						"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
						"User-Agent": "miniooni/0.1.0",
					},
					Method: "GET",
					Tor: model.ArchivalHTTPTor{
						ExitIP:   nil,
						ExitName: nil,
						IsTor:    false,
					},
					Transport: "tcp",
					URL:       "https://www.example.com/",
				},
				Response: model.ArchivalHTTPResponse{
					Body: model.ArchivalScrubbedMaybeBinaryString(
						"Bonsoir, Elliot!",
					),
					BodyIsTruncated: false,
					Code:            200,
					HeadersList: []model.ArchivalHTTPHeader{{
						model.ArchivalScrubbedMaybeBinaryString("Age"),
						model.ArchivalScrubbedMaybeBinaryString("131833"),
					}, {
						model.ArchivalScrubbedMaybeBinaryString("Server"),
						model.ArchivalScrubbedMaybeBinaryString("Apache"),
					}},
					Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
						"Age":    "131833",
						"Server": "Apache",
					},
					Locations: nil,
				},
				T0:            0.7,
				T:             1.33,
				Tags:          []string{"http"},
				TransactionID: 5,
			},
		}, {
			name:      "deserialization of a failed HTTP request",
			input:     []byte(`{"network":"tcp","address":"[2606:2800:220:1:248:1893:25c8:1946]:443","alpn":"h2","failure":"generic_timeout_error","request":{"body":"","body_is_truncated":false,"headers_list":[["Accept","text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"],["User-Agent","miniooni/0.1.0"]],"headers":{"Accept":"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8","User-Agent":"miniooni/0.1.0"},"method":"GET","tor":{"exit_ip":null,"exit_name":null,"is_tor":false},"x_transport":"tcp","url":"https://www.example.com/"},"response":{"body":"","body_is_truncated":false,"code":0,"headers_list":[],"headers":{}},"t0":0.4,"t":1.563,"tags":["http"],"transaction_id":6}`),
			expectErr: nil,
			expectStruct: model.ArchivalHTTPRequestResult{
				Network: "tcp",
				Address: "[2606:2800:220:1:248:1893:25c8:1946]:443",
				ALPN:    "h2",
				Failure: (func() *string {
					s := netxlite.FailureGenericTimeoutError
					return &s
				})(),
				Request: model.ArchivalHTTPRequest{
					Body:            model.ArchivalScrubbedMaybeBinaryString(""),
					BodyIsTruncated: false,
					HeadersList: []model.ArchivalHTTPHeader{{
						model.ArchivalScrubbedMaybeBinaryString("Accept"),
						model.ArchivalScrubbedMaybeBinaryString("text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"),
					}, {
						model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
						model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
					}},
					Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{
						"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
						"User-Agent": "miniooni/0.1.0",
					},
					Method: "GET",
					Tor: model.ArchivalHTTPTor{
						ExitIP:   nil,
						ExitName: nil,
						IsTor:    false,
					},
					Transport: "tcp",
					URL:       "https://www.example.com/",
				},
				Response: model.ArchivalHTTPResponse{
					Body:            model.ArchivalScrubbedMaybeBinaryString(""),
					BodyIsTruncated: false,
					Code:            0,
					HeadersList:     []model.ArchivalHTTPHeader{},
					Headers:         map[string]model.ArchivalScrubbedMaybeBinaryString{},
					Locations:       nil,
				},
				T0:            0.4,
				T:             1.563,
				Tags:          []string{"http"},
				TransactionID: 6,
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// parse the JSON
				var data model.ArchivalHTTPRequestResult
				err := json.Unmarshal(tc.input, &data)

				t.Log("got this error", err)
				t.Logf("got this struct %+v", data)

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

				// make sure the deserialization is OK
				if diff := cmp.Diff(tc.expectStruct, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

// This test ensures that ArchivalNetworkEvent is WAI
func TestArchivalNetworkEvent(t *testing.T) {

	// This test ensures that we correctly serialize to JSON.
	t.Run("MarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the input struct
			input model.ArchivalNetworkEvent

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectData is the data we expect to see
			expectData []byte
		}

		cases := []testcase{{
			name: "serialization of a successful network event",
			input: model.ArchivalNetworkEvent{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				NumBytes:      32768,
				Operation:     "read",
				Proto:         "tcp",
				T0:            1.1,
				T:             1.55,
				TransactionID: 77,
				Tags:          []string{"net"},
			},
			expectErr:  nil,
			expectData: []byte(`{"address":"8.8.8.8:443","failure":null,"num_bytes":32768,"operation":"read","proto":"tcp","t0":1.1,"t":1.55,"transaction_id":77,"tags":["net"]}`),
		}, {
			name: "serialization of a failed network event",
			input: model.ArchivalNetworkEvent{
				Address: "8.8.8.8:443",
				Failure: (func() *string {
					s := netxlite.FailureGenericTimeoutError
					return &s
				})(),
				NumBytes:      0,
				Operation:     "read",
				Proto:         "tcp",
				T0:            1.1,
				T:             7,
				TransactionID: 144,
				Tags:          []string{"net"},
			},
			expectErr:  nil,
			expectData: []byte(`{"address":"8.8.8.8:443","failure":"generic_timeout_error","operation":"read","proto":"tcp","t0":1.1,"t":7,"transaction_id":144,"tags":["net"]}`),
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// serialize to JSON
				data, err := json.Marshal(tc.input)

				t.Log("got this error", err)
				t.Log("got this raw data", data)
				t.Logf("converted to string: %s", string(data))

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

				// make sure the serialization is OK
				if diff := cmp.Diff(tc.expectData, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})

	// This test ensures that we can unmarshal from the JSON representation
	t.Run("UnmarshalJSON", func(t *testing.T) {
		// testcase is a test case defined by this function
		type testcase struct {
			// name is the name of the test case
			name string

			// input is the binary input
			input []byte

			// expectErr is the error we expect to see or nil
			expectErr error

			// expectStruct is the struct we expect to see
			expectStruct model.ArchivalNetworkEvent
		}

		cases := []testcase{{
			name:      "deserialization of a successful network event",
			expectErr: nil,
			input:     []byte(`{"address":"8.8.8.8:443","failure":null,"num_bytes":32768,"operation":"read","proto":"tcp","t0":1.1,"t":1.55,"transaction_id":77,"tags":["net"]}`),
			expectStruct: model.ArchivalNetworkEvent{
				Address:       "8.8.8.8:443",
				Failure:       nil,
				NumBytes:      32768,
				Operation:     "read",
				Proto:         "tcp",
				T0:            1.1,
				T:             1.55,
				TransactionID: 77,
				Tags:          []string{"net"},
			},
		}, {
			name:      "deserialization of a failed network event",
			input:     []byte(`{"address":"8.8.8.8:443","failure":"generic_timeout_error","operation":"read","proto":"tcp","t0":1.1,"t":7,"transaction_id":144,"tags":["net"]}`),
			expectErr: nil,
			expectStruct: model.ArchivalNetworkEvent{
				Address: "8.8.8.8:443",
				Failure: (func() *string {
					s := netxlite.FailureGenericTimeoutError
					return &s
				})(),
				NumBytes:      0,
				Operation:     "read",
				Proto:         "tcp",
				T0:            1.1,
				T:             7,
				TransactionID: 144,
				Tags:          []string{"net"},
			},
		}}

		for _, tc := range cases {
			t.Run(tc.name, func(t *testing.T) {
				// parse the JSON
				var data model.ArchivalNetworkEvent
				err := json.Unmarshal(tc.input, &data)

				t.Log("got this error", err)
				t.Logf("got this struct %+v", data)

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

				// make sure the deserialization is OK
				if diff := cmp.Diff(tc.expectStruct, data); diff != "" {
					t.Fatal(diff)
				}
			})
		}
	})
}

func TestArchivalNewHTTPHeadersList(t *testing.T) {

	// testcase is a test case run by this func
	type testcase struct {
		name   string
		input  http.Header
		expect []model.ArchivalHTTPHeader
	}

	cases := []testcase{{
		name:   "with nil input",
		input:  nil,
		expect: []model.ArchivalHTTPHeader{},
	}, {
		name:   "with empty input",
		input:  map[string][]string{},
		expect: []model.ArchivalHTTPHeader{},
	}, {
		name: "common case",
		input: map[string][]string{
			"Content-Type": {"text/html; charset=utf-8"},
			"Via":          {"a", "b", "c"},
			"User-Agent":   {"miniooni/0.1.0"},
		},
		expect: []model.ArchivalHTTPHeader{{
			model.ArchivalScrubbedMaybeBinaryString("Content-Type"),
			model.ArchivalScrubbedMaybeBinaryString("text/html; charset=utf-8"),
		}, {
			model.ArchivalScrubbedMaybeBinaryString("User-Agent"),
			model.ArchivalScrubbedMaybeBinaryString("miniooni/0.1.0"),
		}, {
			model.ArchivalScrubbedMaybeBinaryString("Via"),
			model.ArchivalScrubbedMaybeBinaryString("a"),
		}, {
			model.ArchivalScrubbedMaybeBinaryString("Via"),
			model.ArchivalScrubbedMaybeBinaryString("b"),
		}, {
			model.ArchivalScrubbedMaybeBinaryString("Via"),
			model.ArchivalScrubbedMaybeBinaryString("c"),
		}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := model.ArchivalNewHTTPHeadersList(tc.input)
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestArchivalNewHTTPHeadersMap(t *testing.T) {

	// testcase is a test case run by this func
	type testcase struct {
		name   string
		input  http.Header
		expect map[string]model.ArchivalScrubbedMaybeBinaryString
	}

	cases := []testcase{{
		name:   "with nil input",
		input:  nil,
		expect: map[string]model.ArchivalScrubbedMaybeBinaryString{},
	}, {
		name:   "with empty input",
		input:  map[string][]string{},
		expect: map[string]model.ArchivalScrubbedMaybeBinaryString{},
	}, {
		name: "common case",
		input: map[string][]string{
			"Content-Type": {"text/html; charset=utf-8"},
			"Via":          {"a", "b", "c"},
			"User-Agent":   {"miniooni/0.1.0"},
		},
		expect: map[string]model.ArchivalScrubbedMaybeBinaryString{
			"Content-Type": "text/html; charset=utf-8",
			"Via":          "a",
			"User-Agent":   "miniooni/0.1.0",
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := model.ArchivalNewHTTPHeadersMap(tc.input)
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
