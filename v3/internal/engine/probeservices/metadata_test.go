package probeservices_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
)

func TestValid(t *testing.T) {
	t.Run("fail on probe_cc", func(t *testing.T) {
		var m probeservices.Metadata
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on probe_asn", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC: "IT",
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on platform", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:  "IT",
			ProbeASN: "AS1234",
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on software_name", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:  "IT",
			ProbeASN: "AS1234",
			Platform: "linux",
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on software_version", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:      "IT",
			ProbeASN:     "AS1234",
			Platform:     "linux",
			SoftwareName: "miniooni",
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on supported_tests", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:         "IT",
			ProbeASN:        "AS1234",
			Platform:        "linux",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("fail on missing device_token", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:         "IT",
			ProbeASN:        "AS1234",
			Platform:        "ios",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
			SupportedTests:  []string{"web_connectivity"},
		}
		if m.Valid() != false {
			t.Fatal("expected false here")
		}
	})
	t.Run("success for desktop", func(t *testing.T) {
		m := probeservices.Metadata{
			ProbeCC:         "IT",
			ProbeASN:        "AS1234",
			Platform:        "linux",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
			SupportedTests:  []string{"web_connectivity"},
		}
		if m.Valid() != true {
			t.Fatal("expected true here")
		}
	})
	t.Run("success for mobile", func(t *testing.T) {
		m := probeservices.Metadata{
			DeviceToken:     "xx-xxx-xx-xxxx",
			ProbeCC:         "IT",
			ProbeASN:        "AS1234",
			Platform:        "android",
			SoftwareName:    "miniooni",
			SoftwareVersion: "0.1.0-dev",
			SupportedTests:  []string{"web_connectivity"},
		}
		if m.Valid() != true {
			t.Fatal("expected true here")
		}
	})
}
