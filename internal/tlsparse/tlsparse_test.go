package tlsparse

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

// TLSHandshakeBytes13 contains a TLSv1.3 handshake obtained
// from https://tls13.xargs.org/#client-hello.
var TLSHandshakeBytes13 = []byte{
	// [0:5] record header
	0x16, 0x03, 0x01, 0x00, 0xf8,

	// [5:9] handshake header
	0x01, 0x00, 0x00, 0xf4,

	// [9:11] client version
	0x03, 0x03,

	// [11:43] client random
	0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07,
	0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f,
	0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17,
	0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, 0x1e, 0x1f,

	// [43:44] session ID length
	0x20,

	// [44:76] session ID
	0xe0, 0xe1, 0xe2, 0xe3, 0xe4, 0xe5, 0xe6, 0xe7,
	0xe8, 0xe9, 0xea, 0xeb, 0xec, 0xed, 0xee, 0xef,
	0xf0, 0xf1, 0xf2, 0xf3, 0xf4, 0xf5, 0xf6, 0xf7,
	0xf8, 0xf9, 0xfa, 0xfb, 0xfc, 0xfd, 0xfe, 0xff,

	// [76:78] cipher suites length
	0x00, 0x08,

	// [78:86] cipher suites
	0x13, 0x02, 0x13, 0x03, 0x13, 0x01, 0x00, 0xff,

	// [86:87] legacy compression methods length
	0x01,

	// [87:88] legacy compression methods
	0x00,

	// [88:90] extensions length
	0x00, 0xa3,

	// [90:253] extensions
	// [90:118] server name extension
	// [90:92] type
	0x00, 0x00,
	// [92:94] length
	0x00, 0x18,
	// [94:118] value
	0x00, 0x16, 0x00, 0x00, 0x13, 0x65, 0x78, 0x61,
	0x6d, 0x70, 0x6c, 0x65, 0x2e, 0x75, 0x6c, 0x66,
	0x68, 0x65, 0x69, 0x6d, 0x2e, 0x6e, 0x65, 0x74,

	// [118:126] ec point formats extension
	// [118:120] type
	0x00, 0x0b,
	// [120:122] length
	0x00, 0x04,
	// [122:126] value
	0x03, 0x00, 0x01, 0x02,

	// [126:152] supported groups ext
	// [126:128] type
	0x00, 0x0a,
	// [128:130] length
	0x00, 0x16,
	// [130:152] value
	0x00, 0x14, 0x00, 0x1d, 0x00, 0x17, 0x00, 0x1e,
	0x00, 0x19, 0x00, 0x18, 0x01, 0x00, 0x01, 0x01,
	0x01, 0x02, 0x01, 0x03, 0x01, 0x04,

	// [152:156] session ticket ext
	// [152:154] type
	0x00, 0x23,
	// [154:156] length
	0x00, 0x00,

	// [156:160] encrypt-then-MAC ext
	// [156:158] type
	0x00, 0x16,
	// [158:160] length
	0x00, 0x00,

	// [160:164] extended master secret ext
	// [160:162] type
	0x00, 0x17,
	// [162:164] length
	0x00, 0x00,

	// [164:198] signature algorithsm ext
	// [164:166] type
	0x00, 0x0d,
	// [166:168] length
	0x00, 0x1e,
	// [168:198] value
	0x00, 0x1c, 0x04, 0x03, 0x05, 0x03, 0x06, 0x03,
	0x08, 0x07, 0x08, 0x08, 0x08, 0x09, 0x08, 0x0a,
	0x08, 0x0b, 0x08, 0x04, 0x08, 0x05, 0x08, 0x06,
	0x04, 0x01, 0x05, 0x01, 0x06, 0x01,

	// [198:205] supported versions ext
	// [198:200] type
	0x00, 0x2b,
	// [200:202] length
	0x00, 0x03,
	// [202:205] value
	0x02, 0x03, 0x04,

	// [205:211] PSK key exchange modes ext
	// [205:207] type
	0x00, 0x2d,
	// [207:209] length
	0x00, 0x02,
	// [209:211] value
	0x01, 0x01,

	// [211:253] key share ext
	// [211:213] type
	0x00, 0x33,
	// [213:215] length
	0x00, 0x26,
	// [215:253]
	0x00, 0x24, 0x00, 0x1d, 0x00, 0x20, 0x35, 0x80,
	0x72, 0xd6, 0x36, 0x58, 0x80, 0xd1, 0xae, 0xea,
	0x32, 0x9a, 0xdf, 0x91, 0x21, 0x38, 0x38, 0x51,
	0xed, 0x21, 0xa2, 0x8e, 0x3b, 0x75, 0xe9, 0x65,
	0xd0, 0xd2, 0xcd, 0x16, 0x62, 0x54,
}

func TestTLSHandshakesLength(t *testing.T) {
	t.Run("for TLSv1.3 handshake", func(t *testing.T) {
		if length := len(TLSHandshakeBytes13); length != 253 {
			t.Fatal("expected 253 got", length)
		}
	})
}

func TestUnmarshalRecordHeader(t *testing.T) {

	type testcase struct {
		// name is the test case name
		name string

		// rawInput is the raw input to use
		rawInput []byte

		// expectRecordHeader is the expected record header
		expectRecordHeader *RecordHeader

		// expectRest is what we expect rest to contain
		expectRest []byte

		// expectErr is the expected error
		expectErr error
	}

	var testcases = []testcase{{
		name:     "for TLSv1.3 handshake",
		rawInput: TLSHandshakeBytes13,
		expectRecordHeader: &RecordHeader{
			ContentType:     0x16,
			ProtocolVersion: 0x0301,
			Rest:            TLSHandshakeBytes13[5:],
		},
		expectRest: nil,
		expectErr:  nil,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			rh, rest, err := UnmarshalRecordHeader(tc.rawInput)

			// check the error
			switch {
			case err == nil && tc.expectErr == nil:
			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			default:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
			}

			// check whether rest is empty
			switch {
			case len(tc.expectRest) <= 0 && rest.Empty():
			case len(tc.expectRest) > 0 && !rest.Empty():
				if diff := cmp.Diff(tc.expectRest, rest); diff != "" {
					t.Fatal(diff)
				}
			default:
				t.Fatal("expected", tc.expectRest, "got", rest.Empty())
			}

			// check the result
			if diff := cmp.Diff(tc.expectRecordHeader, rh); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestUnmarshalHandshakeHeader(t *testing.T) {

	type testcase struct {
		// name is the test case name
		name string

		// rawInput is the raw input to use
		rawInput []byte

		// expectHandshake is the expected handshake
		expectHandshake *Handshake

		// expectErr is the expected error
		expectErr error
	}

	var testcases = []testcase{{
		name:     "for TLSv1.3 handshake",
		rawInput: TLSHandshakeBytes13[5:],
		expectHandshake: &Handshake{
			HandshakeType: 0x01,
			ClientHello: &ClientHello{
				ProtocolVersion:          0x0303,
				Random:                   TLSHandshakeBytes13[11:43],
				LegacySessionID:          TLSHandshakeBytes13[44:76],
				CipherSuites:             TLSHandshakeBytes13[78:86],
				LegacyCompressionMethods: TLSHandshakeBytes13[87:88],
				Extensions:               TLSHandshakeBytes13[90:253],
			},
		},
		expectErr: nil,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			hx, err := UnmarshalHandshake(tc.rawInput)

			// check the error
			switch {
			case err == nil && tc.expectErr == nil:
			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			default:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
			}

			// check the result
			if diff := cmp.Diff(tc.expectHandshake, hx); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestUnmarshalExtensions(t *testing.T) {

	type testcase struct {
		// name is the test case name
		name string

		// rawInput is the raw input to use
		rawInput []byte

		// expectExtensions is the expected extensions
		expectExtensions []*Extension

		// expectErr is the expected error
		expectErr error
	}

	var testcases = []testcase{{
		name:     "for TLSv1.3 handshake",
		rawInput: TLSHandshakeBytes13[90:253],
		expectExtensions: []*Extension{{
			Type: 0,
			Data: TLSHandshakeBytes13[94:118],
		}, {
			Type: 11,
			Data: TLSHandshakeBytes13[122:126],
		}, {
			Type: 10,
			Data: TLSHandshakeBytes13[130:152],
		}, {
			Type: 35,
			Data: []byte{},
		}, {
			Type: 22,
			Data: []byte{},
		}, {
			Type: 23,
			Data: []byte{},
		}, {
			Type: 13,
			Data: TLSHandshakeBytes13[168:198],
		}, {
			Type: 43,
			Data: TLSHandshakeBytes13[202:205],
		}, {
			Type: 45,
			Data: TLSHandshakeBytes13[209:211],
		}, {
			Type: 51,
			Data: TLSHandshakeBytes13[215:253],
		}},
		expectErr: nil,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			exts, err := UnmarshalExtensions(tc.rawInput)

			// check the error
			switch {
			case err == nil && tc.expectErr == nil:
			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			default:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
			}

			// check the result
			if diff := cmp.Diff(tc.expectExtensions, exts); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestFindServerNameExtension(t *testing.T) {

	type testcase struct {
		// name is the test case name
		name string

		// rawInput is the raw input to use
		rawInput []byte

		// expectExtension is the expected extension
		expectExtension *Extension

		// expectFound is whether we expect to find it
		expectFound bool
	}

	var testcases = []testcase{{
		name:     "for TLSv1.3 handshake",
		rawInput: TLSHandshakeBytes13[90:253],
		expectExtension: &Extension{
			Type: 0,
			Data: TLSHandshakeBytes13[94:118],
		},
		expectFound: true,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			exts, err := UnmarshalExtensions(tc.rawInput)
			if err != nil {
				t.Fatal(err)
			}

			ext, good := FindServerNameExtension(exts)
			switch {
			case tc.expectFound && good:
			case !tc.expectFound && !good:
			default:
				t.Fatal("expected", tc.expectFound, "got", good)
			}

			// check the result
			if diff := cmp.Diff(tc.expectExtension, ext); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestUnmarshalServerNameExtension(t *testing.T) {

	type testcase struct {
		// name is the test case name
		name string

		// rawInput is the raw input to use
		rawInput []byte

		// expectSNI is the expected extensions
		expectSNI string

		// expectErr is the expected error
		expectErr error
	}

	var testcases = []testcase{{
		name:      "for TLSv1.3 handshake",
		rawInput:  TLSHandshakeBytes13[94:118],
		expectSNI: "example.ulfheim.net",
		expectErr: nil,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			sni, err := UnmarshalServerNameExtension(tc.rawInput)

			// check the error
			switch {
			case err == nil && tc.expectErr == nil:
			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			default:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
			}

			// check the result
			if diff := cmp.Diff(tc.expectSNI, sni); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
