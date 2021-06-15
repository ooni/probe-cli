package humanize

import "testing"

func TestGood(t *testing.T) {
	if v := SI(128, "bit/s"); v != "128.00  bit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(1280, "bit/s"); v != "  1.28 kbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(12800, "bit/s"); v != " 12.80 kbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(128000, "bit/s"); v != "128.00 kbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(1280000, "bit/s"); v != "  1.28 Mbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(12800000, "bit/s"); v != " 12.80 Mbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(128000000, "bit/s"); v != "128.00 Mbit/s" {
		t.Fatal("unexpected result", v)
	}
	if v := SI(1280000000, "bit/s"); v != "  1.28 Gbit/s" {
		t.Fatal("unexpected result", v)
	}
}
