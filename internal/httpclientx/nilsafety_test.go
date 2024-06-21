package httpclientx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNilSafetyErrorIfNil(t *testing.T) {

	// testcase is a test case implemented by this function.
	type testcase struct {
		name   string
		input  any
		err    error
		output any
	}

	cases := []testcase{{
		name: "with a nil map",
		input: func() any {
			var v map[string]string
			return v
		}(),
		err:    ErrIsNil,
		output: nil,
	}, {
		name:   "with a non-nil but empty map",
		input:  make(map[string]string),
		err:    nil,
		output: make(map[string]string),
	}, {
		name:   "with a non-nil non-empty map",
		input:  map[string]string{"a": "b"},
		err:    nil,
		output: map[string]string{"a": "b"},
	}, {
		name: "with a nil pointer",
		input: func() any {
			var v *apiRequest
			return v
		}(),
		err:    ErrIsNil,
		output: nil,
	}, {
		name:   "with a non-nil empty pointer",
		input:  &apiRequest{},
		err:    nil,
		output: &apiRequest{},
	}, {
		name:   "with a non-nil non-empty pointer",
		input:  &apiRequest{UserID: 11},
		err:    nil,
		output: &apiRequest{UserID: 11},
	}, {
		name: "with a nil slice",
		input: func() any {
			var v []int
			return v
		}(),
		err:    ErrIsNil,
		output: nil,
	}, {
		name:   "with a non-nil empty slice",
		input:  []int{},
		err:    nil,
		output: []int{},
	}, {
		name:   "with a non-nil non-empty slice",
		input:  []int{44},
		err:    nil,
		output: []int{44},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			output, err := NilSafetyErrorIfNil(tc.input)

			switch {
			case err == nil && tc.err == nil:
				if diff := cmp.Diff(tc.output, output); diff != "" {
					t.Fatal(diff)
				}
				return

			case err != nil && tc.err == nil:
				t.Fatal("expected", tc.err.Error(), "got", err.Error())
				return

			case err == nil && tc.err != nil:
				t.Fatal("expected", tc.err.Error(), "got", err.Error())
				return

			case err != nil && tc.err != nil:
				if err.Error() != tc.err.Error() {
					t.Fatal("expected", tc.err.Error(), "got", err.Error())
				}
				return
			}
		})
	}
}

func TestNilSafetyAvoidNilByteSlice(t *testing.T) {
	t.Run("for nil byte slice", func(t *testing.T) {
		output := NilSafetyAvoidNilBytesSlice(nil)
		if output == nil {
			t.Fatal("expected non-nil")
		}
		if len(output) != 0 {
			t.Fatal("expected zero length")
		}
	})

	t.Run("for non-nil byte slice", func(t *testing.T) {
		expected := []byte{44}
		output := NilSafetyAvoidNilBytesSlice(expected)
		if diff := cmp.Diff(expected, output); diff != "" {
			t.Fatal("not the same pointer")
		}
	})
}
