package connid

import "testing"

func TestTCP(t *testing.T) {
	num := Compute("tcp", "1.2.3.4:6789")
	if num != 6789 {
		t.Fatal("unexpected result")
	}
}

func TestTCP4(t *testing.T) {
	num := Compute("tcp4", "130.192.91.211:34566")
	if num != 34566 {
		t.Fatal("unexpected result")
	}
}

func TestTCP6(t *testing.T) {
	num := Compute("tcp4", "[::1]:4444")
	if num != 4444 {
		t.Fatal("unexpected result")
	}
}

func TestUDP(t *testing.T) {
	num := Compute("udp", "1.2.3.4:6789")
	if num != -6789 {
		t.Fatal("unexpected result")
	}
}

func TestUDP4(t *testing.T) {
	num := Compute("udp4", "130.192.91.211:34566")
	if num != -34566 {
		t.Fatal("unexpected result")
	}
}

func TestUDP6(t *testing.T) {
	num := Compute("udp6", "[::1]:4444")
	if num != -4444 {
		t.Fatal("unexpected result")
	}
}

func TestInvalidAddress(t *testing.T) {
	num := Compute("udp6", "[::1]")
	if num != 0 {
		t.Fatal("unexpected result")
	}
}

func TestInvalidPort(t *testing.T) {
	num := Compute("udp6", "[::1]:antani")
	if num != 0 {
		t.Fatal("unexpected result")
	}
}

func TestNegativePort(t *testing.T) {
	num := Compute("udp6", "[::1]:-1")
	if num != 0 {
		t.Fatal("unexpected result")
	}
}

func TestLargePort(t *testing.T) {
	num := Compute("udp6", "[::1]:65536")
	if num != 0 {
		t.Fatal("unexpected result")
	}
}

func TestInvalidNetwork(t *testing.T) {
	num := Compute("unix", "[::1]:65531")
	if num != 0 {
		t.Fatal("unexpected result")
	}
}
