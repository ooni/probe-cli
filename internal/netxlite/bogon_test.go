package netxlite

import "testing"

func TestIsBogon(t *testing.T) {
	if IsBogon("antani") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("127.0.0.1") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("1.1.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("10.0.1.1") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("::1") != true {
		t.Fatal("unexpected result")
	}
}
