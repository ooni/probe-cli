package webconnectivitylte

import "testing"

func TestNumRedirects(t *testing.T) {
	const count = 10
	nr := NewNumRedirects(count)
	for idx := 0; idx < count; idx++ {
		if !nr.CanFollowOneMoreRedirect() {
			t.Fatal("got false with idx=", idx)
		}
	}
	if nr.CanFollowOneMoreRedirect() {
		t.Fatal("got true after the loop")
	}
}
