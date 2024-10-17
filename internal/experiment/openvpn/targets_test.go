package openvpn

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func Test_pickFromDefaultOONIOpenVPNConfig(t *testing.T) {
	pick := pickFromDefaultOONIOpenVPNConfig()

	if pick.Cipher != "AES-256-GCM" {
		t.Fatal("cipher unexpected")
	}
	if pick.SafeCA != defaultCA {
		t.Fatal("ca unexpected")
	}
}

func TestSampleN(t *testing.T) {
	// Table of test cases
	tests := []struct {
		name     string
		a        []string
		n        int
		expected int // Expected length of result
	}{
		{"n less than slice length", []string{"a", "b", "c", "d", "e"}, 3, 3},
		{"n greater than slice length", []string{"a", "b", "c", "d", "e"}, 10, 5},
		{"n equal to zero", []string{"a", "b", "c", "d", "e"}, 0, 0},
		{"empty slice", []string{}, 3, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sampleN(tt.a, tt.n)

			// Check the length of the result
			if len(result) != tt.expected {
				t.Errorf("Expected %d items, got %d", tt.expected, len(result))
			}

			// Check for duplicates
			seen := make(map[string]struct{})
			for _, v := range result {
				if _, exists := seen[v]; exists {
					t.Errorf("Duplicate value %s found", v)
				}
				seen[v] = struct{}{}
			}
		})
	}
}

func Test_resolveOONIAddresses(t *testing.T) {
	expected := []string{
		"108.61.164.186",
		"37.218.243.98",
	}
	t.Run("check resolution is what we expect", func(t *testing.T) {
		if testing.Short() {
			// this test uses the real internet so we want to skip this in short mode
			t.Skip("skip test in short mode")
		}

		got, err := resolveOONIAddresses(model.DiscardLogger)
		if err != nil {
			t.Errorf("resolveOONIAddresses() error = %v, wantErr %v", err, nil)
			return
		}
		if diff := cmp.Diff(expected, got, cmpopts.SortSlices(func(x, y string) bool { return x < y })); diff != "" {
			t.Errorf("Mismatch (-expected +got):\n%s", diff)
		}
	})
}
