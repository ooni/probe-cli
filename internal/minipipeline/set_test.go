package minipipeline

import (
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/must"
)

func TestSet(t *testing.T) {
	t.Run("NewSet", func(t *testing.T) {
		set := NewSet[int64](11, 17, 114, 117)
		expect := map[int64]bool{11: true, 17: true, 114: true, 117: true}
		if diff := cmp.Diff(expect, set.state); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Add", func(t *testing.T) {
		var set Set[int64]
		set.Add(11, 17, 114, 117)
		expect := map[int64]bool{11: true, 17: true, 114: true, 117: true}
		if diff := cmp.Diff(expect, set.state); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Len", func(t *testing.T) {
		var set Set[int64]
		set.Add(11, 17, 114, 117)
		if length := set.Len(); length != 4 {
			t.Fatal("expected 4 but got", length)
		}
	})

	t.Run("Remove", func(t *testing.T) {
		var set Set[int64]
		set.Add(11, 17, 114, 117)
		set.Remove(114)
		expect := map[int64]bool{11: true, 17: true, 117: true}
		if diff := cmp.Diff(expect, set.state); diff != "" {
			t.Fatal(diff)
		}
		if length := set.Len(); length != 3 {
			t.Fatal("expected 3 but got", length)
		}
	})

	t.Run("Keys", func(t *testing.T) {
		t.Run("with empty set", func(t *testing.T) {
			var set Set[int64]
			if length := len(set.Keys()); length != 0 {
				t.Fatal("expected zero but got", length)
			}
			if a, b := len(set.Keys()), set.Len(); a != b {
				t.Fatal("len(set.Keys()) =", a, "but set.Len() =", b)
			}
		})

		t.Run("with entries", func(t *testing.T) {
			var set Set[int64]
			set.Add(10, 11, 12, 13)
			if length := len(set.Keys()); length != 4 {
				t.Fatal("expected 4 but got", length)
			}
			if a, b := len(set.Keys()), set.Len(); a != b {
				t.Fatal("len(set.Keys()) =", a, "but set.Len() =", b)
			}
			expect := []int64{10, 11, 12, 13}
			if diff := cmp.Diff(expect, set.Keys()); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("MarshalJSON", func(t *testing.T) {
		var set Set[int64]
		set.Add(10, 11, 12, 13)
		expect := []byte(`[10,11,12,13]`)
		got := must.MarshalJSON(set)
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("UnmarshalJSON", func(t *testing.T) {
		t.Run("with legit input", func(t *testing.T) {
			input := []byte(`[10,11,12,13]`)
			var set Set[int64]
			must.UnmarshalJSON(input, &set)
			expect := map[int64]bool{10: true, 11: true, 12: true, 13: true}
			if diff := cmp.Diff(expect, set.state); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("with unexpected input", func(t *testing.T) {
			input := []byte(`{}`)
			var set Set[int64]
			err := json.Unmarshal(input, &set)
			if err == nil || err.Error() != "json: cannot unmarshal object into Go value of type []int64" {
				t.Fatal("unexpected error", err)
			}
		})
	})

	t.Run("Contains", func(t *testing.T) {
		var set Set[int64]
		set.Add(10, 11, 12, 13)
		if !set.Contains(10) {
			t.Fatal("expected true")
		}
		if set.Contains(117) {
			t.Fatal("expected false")
		}
	})
}
