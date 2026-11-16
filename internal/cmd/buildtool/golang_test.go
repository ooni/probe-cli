package main

import "testing"

func TestGolangVersionCheckP(t *testing.T) {
	t.Run("running newer than the floor passes", func(t *testing.T) {
		if !golangVersionCheckP("1.27.0", "go1.27.4") {
			t.Fatal("expected true")
		}
	})

	t.Run("running equal to the floor passes", func(t *testing.T) {
		if !golangVersionCheckP("1.27.0", "go1.27.0") {
			t.Fatal("expected true")
		}
	})

	t.Run("running older than the floor fails", func(t *testing.T) {
		if golangVersionCheckP("1.27.0", "go1.26.9") {
			t.Fatal("expected false")
		}
	})
}

func TestGolangCheck(t *testing.T) {
	golangCheck()
}

func TestGolangToolchainVersion(t *testing.T) {
	// We expect a non-empty, "go"-stripped version from this module's go.mod.
	if v := golangToolchainVersion(); v == "" || v[0] < '0' || v[0] > '9' {
		t.Fatal("unexpected toolchain version", v)
	}
}
