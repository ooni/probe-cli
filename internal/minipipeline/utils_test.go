package minipipeline

import "testing"

func TestUtilsExtractTagDepth(t *testing.T) {
	t.Run("with nil tags list", func(t *testing.T) {
		result := utilsExtractTagDepth(nil)
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with empty tags list", func(t *testing.T) {
		result := utilsExtractTagDepth([]string{})
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with missing depth=123 tag", func(t *testing.T) {
		result := utilsExtractTagDepth([]string{"a", "b", "c", "d"})
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with depth=NotANumber tag", func(t *testing.T) {
		result := utilsExtractTagDepth([]string{"depth=NotANumber"})
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("we return the last entry", func(t *testing.T) {
		result := utilsExtractTagDepth([]string{"depth=10", "depth=12"})
		if result.IsNone() {
			t.Fatal("expected not none")
		}
		if value := result.Unwrap(); value != 12 {
			t.Fatal("expected 12, got", value)
		}
	})
}

func TestUtilsTagFetchBody(t *testing.T) {
	t.Run("with nil tags list", func(t *testing.T) {
		result := utilsExtractTagFetchBody(nil)
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with empty tags list", func(t *testing.T) {
		result := utilsExtractTagFetchBody([]string{})
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with missing feth_body=BOOL tag", func(t *testing.T) {
		result := utilsExtractTagFetchBody([]string{"a", "b", "c", "d"})
		if !result.IsNone() {
			t.Fatal("expected none")
		}
	})

	t.Run("with fetch_body=false tag", func(t *testing.T) {
		result := utilsExtractTagFetchBody([]string{"fetch_body=false"})
		if result.IsNone() {
			t.Fatal("expected not none")
		}
		if value := result.Unwrap(); value != false {
			t.Fatal("expected false, got", value)
		}
	})

	t.Run("with fetch_body=true tag", func(t *testing.T) {
		result := utilsExtractTagFetchBody([]string{"fetch_body=true"})
		if result.IsNone() {
			t.Fatal("expected not none")
		}
		if value := result.Unwrap(); value != true {
			t.Fatal("expected true, got", value)
		}
	})

	t.Run("we return the last entry", func(t *testing.T) {
		result := utilsExtractTagFetchBody([]string{"fetch_body=false", "fetch_body=true"})
		if result.IsNone() {
			t.Fatal("expected not none")
		}
		if value := result.Unwrap(); value != true {
			t.Fatal("expected false, got", value)
		}
	})
}
