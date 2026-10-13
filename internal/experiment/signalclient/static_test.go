package signalclient

import (
	"net"
	"testing"
)

func TestParseStaticAddresses(t *testing.T) {
	got, err := parseStaticAddresses("chat.signal.org=13.248.212.111, 2600::1 ; CDN.signal.org=1.2.3.4")
	if err != nil {
		t.Fatal(err)
	}
	if len(got["chat.signal.org"]) != 2 || got["chat.signal.org"][1].ip != "2600::1" || got["cdn.signal.org"][0].source != "option" {
		t.Errorf("%+v", got)
	}
	for _, bad := range []string{"chat.signal.org", "chat.signal.org=not-an-ip", "=1.2.3.4"} {
		if _, err := parseStaticAddresses(bad); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestBuiltinStaticAddresses(t *testing.T) {
	all := builtinStaticAddresses()
	v4 := 0
	for _, list := range all {
		for _, a := range list {
			if net.ParseIP(a.ip) == nil {
				t.Errorf("not an IP: %q", a.ip)
			}
			if net.ParseIP(a.ip).To4() != nil {
				v4++
			}
		}
	}
	if v4 != 36 { // the spec's 34 distinct IPv4 addresses, with cdn2 and cdn3 sharing 2
		t.Errorf("%d IPv4 static addresses", v4)
	}
	cdsi := all["cdsi.signal.org"]
	if len(cdsi) != 2 || cdsi[0].source != "libsignal" || cdsi[1].source != "libsignal" {
		t.Errorf("cdsi: %+v (an address keeps the first list that has it)", cdsi)
	}
	got := staticCandidates(all["svr2.signal.org"], "20.236.21.158")
	if len(got) != 3 || got[0].ip != "20.104.52.125" || got[1].ip != "20.9.45.98" || net.ParseIP(got[2].ip).To4() != nil {
		t.Errorf("candidates %+v", got)
	}
	if first, ok := firstStatic(all["updates2.signal.org"]); !ok || first.ip != "104.18.26.44" || first.source != "desktop" {
		t.Errorf("first static %+v", first)
	}
	if _, ok := firstStatic(nil); ok {
		t.Error("first static of an empty list")
	}
}
