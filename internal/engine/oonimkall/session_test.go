package oonimkall

import "testing"

func TestNewCheckInInfoWebConnectivityNilPointer(t *testing.T) {
	out := newCheckInInfoWebConnectivity(nil)
	if out != nil {
		t.Fatal("expected nil pointer")
	}
}
