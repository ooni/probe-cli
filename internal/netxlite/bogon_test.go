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
	if IsBogon("8.8.4.4") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("2001:4860:4860::8844") != false {
		t.Fatal("unexpected result")
	}
	if IsBogon("10.0.1.1") != true {
		t.Fatal("unexpected result")
	}
	if IsBogon("::1") != true {
		t.Fatal("unexpected result")
	}
}

func TestIsLoopback(t *testing.T) {
	if IsLoopback("antani") != true {
		t.Fatal("unexpected result")
	}
	if IsLoopback("127.0.0.1") != true {
		t.Fatal("unexpected result")
	}
	if IsLoopback("1.1.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("8.8.4.4") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("2001:4860:4860::8844") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("10.0.1.1") != false {
		t.Fatal("unexpected result")
	}
	if IsLoopback("::1") != true {
		t.Fatal("unexpected result")
	}
}
