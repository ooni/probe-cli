package signalclient

import (
	"strings"
	"testing"
)

func TestFronts(t *testing.T) {
	f := builtinFronts("EG")
	if f["android_g.chat"].snis[0] != "www.google.com.eg" || len(f["android_g.chat"].snis) != 6 ||
		f["android_g.storage"].pathPrefix != "/storage" || f["reflector_g"].snis[0] != "www.google.com" {
		t.Errorf("EG: %+v", f)
	}
	if f := builtinFronts("BR"); f["android_g.chat"].snis[0] != "www.google.com" {
		t.Errorf("BR: %+v", f["android_g.chat"])
	}
	if err := parseFronts("reflector_f=r.example@a.example,b.example;android_g=h.example@c.example", f); err != nil {
		t.Fatal(err)
	}
	if fc := f["reflector_f"]; fc.hostHeader != "r.example" || strings.Join(fc.snis, ",") != "a.example,b.example" {
		t.Errorf("reflector_f %+v", fc)
	}
	if fc := f["android_g.storage"]; fc.hostHeader != "h.example" || fc.snis[0] != "c.example" || fc.pathPrefix != "/storage" {
		t.Errorf("android_g.storage %+v (the path prefix is kept)", fc)
	}
	for _, bad := range []string{"reflector_f=host", "reflector_x=h@s", "reflector_f.chat=h@s", "android_f=h@", "=h@s"} {
		if err := parseFronts(bad, builtinFronts("")); err == nil {
			t.Errorf("accepted %q", bad)
		}
	}
}

func TestPickFront(t *testing.T) {
	if got := pickFront(RouteAndroidG, "EG", builtinFronts("EG")["android_g.chat"].snis); got != "www.google.com.eg" {
		t.Errorf("android_g EG: %s", got)
	}
	if got := pickFront(RouteAndroidF, "BR", fastlyFronts); got != fastlyFronts[0] {
		t.Errorf("android_f BR: %s", got)
	}
	seen := map[string]bool{}
	for range 200 {
		seen[pickFront(RouteReflectorF, "", fastlyFronts)] = true
		seen[pickFront(RouteAndroidF, "IR", fastlyFronts)] = true
	}
	if len(seen) != len(fastlyFronts) {
		t.Errorf("random fronts never picked some: %v", seen)
	}
}
