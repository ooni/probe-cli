package reflectx

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/testingx"
)

type example struct {
	_    struct{}
	Age  int64
	Ch   chan int64
	F    bool
	Fmp  map[string]*bool
	Fp   *bool
	Fv   []bool
	Fvp  []*bool
	Name string
	Ptr  *int64
	V    []int64
}

var nonzero example

func init() {
	ff := &testingx.FakeFiller{}
	ff.Fill(&nonzero)
}

func TestStructOrStructPtrIsZero(t *testing.T) {

	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the name of the test case
		name string

		// input is the input
		input any

		// expect is the expected result
		expect bool
	}

	cases := []testcase{{
		name:   "[struct] with zero value",
		input:  example{},
		expect: true,
	}, {
		name:   "[ptr] with zero value",
		input:  &example{},
		expect: true,
	}, {
		name:   "[struct] with nonzero value",
		input:  nonzero,
		expect: false,
	}, {
		name:   "[ptr] with nonzero value",
		input:  &nonzero,
		expect: false,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Logf("input: %#v", tc.input)
			if got := StructOrStructPtrIsZero(tc.input); got != tc.expect {
				t.Fatal("expected", tc.expect, "got", got)
			}
		})
	}
}
