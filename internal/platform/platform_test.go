package platform

import (
	"fmt"
	"testing"
)

func TestGood(t *testing.T) {
	var expected bool
	switch Name() {
	case "android", "freebsd", "openbsd", "ios", "linux", "macos", "windows":
		expected = true
	}
	if !expected {
		t.Fatal("unexpected platform name")
	}
}

func TestName(t *testing.T) {
	var runtimevariables = []struct {
		expected string
		goos     string
	}{{
		expected: "android",
		goos:     "android",
	}, {
		expected: "freebsd",
		goos:     "freebsd",
	}, {
		expected: "openbsd",
		goos:     "openbsd",
	}, {
		expected: "ios",
		goos:     "ios",
	}, {
		expected: "linux",
		goos:     "linux",
	}, {
		expected: "macos",
		goos:     "darwin",
	}, {
		expected: "unknown",
		goos:     "solaris",
	}, {
		expected: "windows",
		goos:     "windows",
	}}
	for _, v := range runtimevariables {
		t.Run(fmt.Sprintf("with %s", v.goos), func(t *testing.T) {
			if name(v.goos) != v.expected {
				t.Fatal("unexpected results")
			}
		})
	}
}
