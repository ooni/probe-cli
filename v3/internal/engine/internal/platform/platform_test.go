package platform

import (
	"fmt"
	"testing"
)

func TestGood(t *testing.T) {
	var expected bool
	switch Name() {
	case "android", "ios", "linux", "macos", "windows":
		expected = true
	}
	if !expected {
		t.Fatal("unexpected platform name")
	}
}

func TestPuregoname(t *testing.T) {
	var runtimevariables = []struct {
		expected string
		goarch   string
		goos     string
	}{{
		expected: "android",
		goarch:   "*",
		goos:     "android",
	}, {
		expected: "ios",
		goarch:   "arm64",
		goos:     "darwin",
	}, {
		expected: "ios",
		goarch:   "arm",
		goos:     "darwin",
	}, {
		expected: "linux",
		goarch:   "*",
		goos:     "linux",
	}, {
		expected: "macos",
		goarch:   "amd64",
		goos:     "darwin",
	}, {
		expected: "macos",
		goarch:   "386",
		goos:     "darwin",
	}, {
		expected: "unknown",
		goarch:   "*",
		goos:     "solaris",
	}, {
		expected: "unknown",
		goarch:   "mips",
		goos:     "darwin",
	}, {
		expected: "windows",
		goarch:   "*",
		goos:     "windows",
	}}
	for _, v := range runtimevariables {
		t.Run(fmt.Sprintf("with %s/%s", v.goos, v.goarch), func(t *testing.T) {
			if puregoname(v.goos, v.goarch) != v.expected {
				t.Fatal("unexpected results")
			}
		})
	}
}
