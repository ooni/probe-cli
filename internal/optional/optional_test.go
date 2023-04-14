package optional

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestValue(t *testing.T) {

	// Verify that None creates a Value with an indirect == nil
	t.Run("None works as intended", func(t *testing.T) {
		v := None[int]()
		if v.indirect != nil {
			t.Fatal("should be nil")
		}
	})

	t.Run("Some works as intended", func(t *testing.T) {

		// Verify that Some(value) creates a valid underlying pointer to
		// the value when the wrapped type is not a pointer.
		t.Run("for nonzero nonpointer value", func(t *testing.T) {
			underlying := 12345
			v := Some(underlying)
			if v.indirect == nil || *v.indirect != underlying {
				t.Fatal("unexpected indirect")
			}
		})

		// Verify that Some(value) works for a zero input when the
		// wrapped value is not a pointer.
		t.Run("for zero nonpointer value", func(t *testing.T) {
			underlying := 0
			v := Some(underlying)
			if v.indirect == nil || *v.indirect != underlying {
				t.Fatal("unexpected indirect")
			}
		})

		// Verify that Some(value) correctly creates a pointer to the
		// underlying value when we're wrapping a pointer type
		t.Run("for nonzero pointer value", func(t *testing.T) {
			underlying := 12345
			v := Some(&underlying)
			if v.indirect == nil || *v.indirect == nil || **v.indirect != underlying {
				t.Fatal("unexpected indirect")
			}
		})

		// Verify that Some(nil) creates an empty value when wrapping a pointer
		t.Run("for zero nonpointer value", func(t *testing.T) {
			var underlying *int
			v := Some(underlying)
			if v.indirect != nil {
				t.Fatal("unexpected indirect", *v.indirect)
			}
		})
	})

	t.Run("UnmarshalJSON works as intended", func(t *testing.T) {

		t.Run("for nonpointer type", func(t *testing.T) {

			// When we wrap a nonpointer and the JSON is valid, we expect
			// the underlying value to be correctly populated
			t.Run("with valid JSON input", func(t *testing.T) {
				type config struct {
					UID Value[int64]
				}

				input := []byte(`{"UID":12345}`)
				var state config
				if err := json.Unmarshal(input, &state); err != nil {
					t.Fatal(err)
				}

				if state.UID.indirect == nil || *state.UID.indirect != 12345 {
					t.Fatal("did not set indirect correctly")
				}
			})

			// When the JSON input is incompatible, there should always
			// be an error indicating we cannot assign and obviously the
			// Value should not have been set.
			t.Run("with incompatible JSON input", func(t *testing.T) {
				type config struct {
					UID Value[int64]
				}

				input := []byte(`{"UID":[]}`)
				var state config
				err := json.Unmarshal(input, &state)
				if err == nil || err.Error() != "json: cannot unmarshal array into Go struct field config.UID of type int64" {
					t.Fatal("unexpected err", err)
				}

				if state.UID.indirect != nil {
					t.Fatal("should not have set", *state.UID.indirect)
				}
			})

			// As a special case, when the JSON input is `null`, we should behave
			// like the None constructor had been called.
			t.Run("with null JSON input", func(t *testing.T) {
				type config struct {
					UID Value[int64]
				}

				input := []byte(`{"UID":null}`)
				var state config
				err := json.Unmarshal(input, &state)
				if err != nil {
					t.Fatal(err)
				}

				if state.UID.indirect != nil {
					t.Fatal("should not have set", *state.UID.indirect)
				}
			})
		})

		t.Run("for pointer type", func(t *testing.T) {

			// When the JSON input is valid, we expect that the underlying pointer
			// is a pointer to the expected value.
			t.Run("with valid JSON input", func(t *testing.T) {
				type config struct {
					UID Value[*int64]
				}

				input := []byte(`{"UID":12345}`)
				var state config
				if err := json.Unmarshal(input, &state); err != nil {
					t.Fatal(err)
				}

				if state.UID.indirect == nil || *state.UID.indirect == nil || **state.UID.indirect != 12345 {
					t.Fatal("did not set indirect correctly")
				}
			})

			// With incompatible JSON input, there should be an error and obviously
			// we should not have set any value inside the Value
			t.Run("with incompatible JSON input", func(t *testing.T) {
				type config struct {
					UID Value[*int64]
				}

				input := []byte(`{"UID":[]}`)
				var state config
				err := json.Unmarshal(input, &state)
				if err == nil || err.Error() != "json: cannot unmarshal array into Go struct field config.UID of type int64" {
					t.Fatal("unexpected err", err)
				}

				if state.UID.indirect != nil {
					t.Fatal("should not have set", *state.UID.indirect)
				}
			})

			// When the JSON input is `null`, the code should behave like we
			// had invoked the None constructor for the pointer type.
			t.Run("with null JSON input", func(t *testing.T) {
				type config struct {
					UID Value[*int64]
				}

				input := []byte(`{"UID":null}`)
				var state config
				err := json.Unmarshal(input, &state)
				if err != nil {
					t.Fatal(err)
				}

				if state.UID.indirect != nil {
					t.Fatal("should not have set", *state.UID.indirect)
				}
			})
		})
	})

	t.Run("MarshalJSON works as intended", func(t *testing.T) {
		t.Run("for an empty Value", func(t *testing.T) {
			value := None[int]()
			got, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			expect := []byte(`null`)
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("for an nonempty Value", func(t *testing.T) {
			value := Some(12345)
			got, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			expect := []byte(`12345`)
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("for non-empty concrete type", func(t *testing.T) {
			type config struct {
				UID Value[int] `json:",omitempty"`
			}
			c := &config{
				UID: Some(12345),
			}
			got, err := json.Marshal(c)
			if err != nil {
				t.Fatal(err)
			}
			expect := []byte(`{"UID":12345}`)
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("IsNone works as intended", func(t *testing.T) {
		t.Run("for empty Value", func(t *testing.T) {
			value := None[int]()
			if !value.IsNone() {
				t.Fatal("should be none")
			}
		})

		t.Run("for nonempty Value", func(t *testing.T) {
			value := Some(12345)
			if value.IsNone() {
				t.Fatal("should not be none")
			}
		})
	})

	t.Run("Unwrap works as intended", func(t *testing.T) {
		t.Run("for an empty Value", func(t *testing.T) {
			value := None[int]()
			var err error
			func() {
				defer func() {
					err = recover().(error)
				}()
				out := value.Unwrap()
				t.Log(out)
			}()
			if err == nil || err.Error() != "is none" {
				t.Fatal("unexpected err", err)
			}
		})

		t.Run("for a nonempty Value", func(t *testing.T) {
			value := Some(12345)
			if v := value.Unwrap(); v != 12345 {
				t.Fatal("unexpected value", v)
			}
		})
	})

	t.Run("UnwrapOr works as intended", func(t *testing.T) {
		t.Run("for an empty Value", func(t *testing.T) {
			value := None[int]()
			if v := value.UnwrapOr(555); v != 555 {
				t.Fatal("unexpected value", v)
			}
		})

		t.Run("for a nonempty Value", func(t *testing.T) {
			value := Some(12345)
			if v := value.UnwrapOr(555); v != 12345 {
				t.Fatal("unexpected value", v)
			}
		})
	})
}
