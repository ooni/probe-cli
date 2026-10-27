package oonimkall

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/version"
)

func TestEngineName(t *testing.T) {
	if got := EngineName(); got != "ooniprobe-engine" {
		t.Fatalf("unexpected engine name: got %q, want %q", got, "ooniprobe-engine")
	}
}

func TestEngineVersion(t *testing.T) {
	if got := EngineVersion(); got != version.Version {
		t.Fatalf("unexpected engine version: got %q, want %q", got, version.Version)
	}
}
