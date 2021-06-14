package humanize

import "testing"

func TestGood(t *testing.T) {
	if SI(128, "bit/s") != "128  bit/s" {
		t.Fatal("unexpected result")
	}
	if SI(1280, "bit/s") != "  1 kbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(12800, "bit/s") != " 13 kbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(128000, "bit/s") != "128 kbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(1280000, "bit/s") != "  1 Mbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(12800000, "bit/s") != " 13 Mbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(128000000, "bit/s") != "128 Mbit/s" {
		t.Fatal("unexpected result")
	}
	if SI(1280000000, "bit/s") != "  1 Gbit/s" {
		t.Fatal("unexpected result")
	}
}
