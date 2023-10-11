package webconnectivityqa

import "testing"

func TestAllTestCases(t *testing.T) {
	t.Run("we have at least one test case to run", func(t *testing.T) {
		if len(AllTestCases()) < 1 {
			t.Fatal("expected at least a single test case")
		}
	})
}
