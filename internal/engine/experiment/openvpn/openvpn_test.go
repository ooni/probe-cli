package openvpn

import (
	"testing"
)

func TestExperimentNameAndVersion(t *testing.T) {
	m := NewExperimentMeasurer(Config{})
	if m.ExperimentName() != "openvpn" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.0.1" {
		t.Fatal("invalid experiment version")
	}
}
