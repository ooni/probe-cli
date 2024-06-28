package experimentconfig

import (
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestDefaultOptionsSerializer(t *testing.T) {
	// configuration is the configuration we're testing the serialization of.
	//
	// Note that there's no `ooni:"..."` annotation here because we have changed
	// our model in https://github.com/ooni/probe-cli/pull/1629, and now this kind
	// of annotations are only command-line related.
	type configuration struct {
		// booleans
		ValBool bool

		// integers
		ValInt   int
		ValInt8  int8
		ValInt16 int16
		ValInt32 int32
		ValInt64 int64

		// unsigned integers
		ValUint   uint
		ValUint8  uint8
		ValUint16 uint16
		ValUint32 uint32
		ValUint64 uint64

		// floats
		ValFloat32 float32
		ValFloat64 float64

		// strings
		ValString string

		// unexported fields we should ignore
		privateInt    int
		privateString string
		privateList   []int16

		// safe fields we should ignore
		SafeBool   bool
		SafeInt    int
		SafeString string

		// non-scalar fields we should ignore
		NSList []int64
		NSMap  map[string]string
	}

	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// config is the config to transform into a list of options
		config any

		// expectConfigType is an extra check to make sure we're actually
		// passing the correct type for the config, which is here to ensure
		// that, with a nil pointer to struct, we're not crashing. We need
		// some extra case here because of how the Go type system work,
		// and specifically we want to be sure we're passing an any containing
		// a tuple like (type=*configuration,value=nil).
		//
		// See https://codefibershq.com/blog/golang-why-nil-is-not-always-nil
		expectConfigType string

		// expect is the expected result
		expect []string
	}

	cases := []testcase{
		{
			name:             "we return a nil list for zero values",
			expectConfigType: "*experimentconfig.configuration",
			config:           &configuration{},
			expect:           nil,
		},

		{
			name:             "we return a nil list for non-pointers",
			expectConfigType: "experimentconfig.configuration",
			config:           configuration{},
			expect:           nil,
		},

		{
			name:             "we return a nil list for non-struct pointers",
			expectConfigType: "*int64",
			config: func() *int64 {
				v := int64(12345)
				return &v
			}(),
			expect: nil,
		},

		{
			name:             "we return a nil list for a nil struct pointer",
			expectConfigType: "*experimentconfig.configuration",
			config: func() *configuration {
				return (*configuration)(nil)
			}(),
			expect: nil,
		},

		{
			name:             "we only serialize the fields that should be exported",
			expectConfigType: "*experimentconfig.configuration",
			config: &configuration{
				ValBool:       true,
				ValInt:        1,
				ValInt8:       2,
				ValInt16:      3,
				ValInt32:      4,
				ValInt64:      5,
				ValUint:       6,
				ValUint8:      7,
				ValUint16:     8,
				ValUint32:     9,
				ValUint64:     10,
				ValFloat32:    11,
				ValFloat64:    12,
				ValString:     "tredici",
				privateInt:    14,
				privateString: "quindici",
				privateList:   []int16{16},
				SafeBool:      true,
				SafeInt:       18,
				SafeString:    "diciannove",
				NSList:        []int64{20},
				NSMap:         map[string]string{"21": "22"},
			},
			expect: []string{
				"ValBool=true",
				"ValInt=1",
				"ValInt8=2",
				"ValInt16=3",
				"ValInt32=4",
				"ValInt64=5",
				"ValUint=6",
				"ValUint8=7",
				"ValUint16=8",
				"ValUint32=9",
				"ValUint64=10",
				"ValFloat32=11",
				"ValFloat64=12",
				"ValString=tredici",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// first make sure that tc.config has really the expected
			// type for the reason explained in its docstring
			if actual := fmt.Sprintf("%T", tc.config); actual != tc.expectConfigType {
				t.Fatal("expected", tc.expectConfigType, "got", actual)
			}

			// then serialize the content of the config to a list of strings
			got := DefaultOptionsSerializer(tc.config)

			// finally, make sure that the result matches expectations
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
