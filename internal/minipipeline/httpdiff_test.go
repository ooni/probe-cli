package minipipeline

import "testing"

func TestComputeHTTPDiffStatusCodeMatch(t *testing.T) {
	t.Run(
		"when the control status code is not 2xx and the measurement is different from the control",
		func(t *testing.T) {
			result := ComputeHTTPDiffStatusCodeMatch(200, 500)
			if !result.IsNone() {
				t.Fatal("should be none")
			}
		})

	t.Run("when both are 500", func(t *testing.T) {
		result := ComputeHTTPDiffStatusCodeMatch(500, 500)
		if result.IsNone() {
			t.Fatal("should not be none")
		}
		if result.Unwrap() != true {
			t.Fatal("result should be true")
		}
	})

	t.Run("when both are 200", func(t *testing.T) {
		result := ComputeHTTPDiffStatusCodeMatch(200, 200)
		if result.IsNone() {
			t.Fatal("should not be none")
		}
		if result.Unwrap() != true {
			t.Fatal("result should be true")
		}
	})
}
