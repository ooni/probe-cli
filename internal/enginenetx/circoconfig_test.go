package enginenetx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestCircoConfig(t *testing.T) {
	t.Run("NewCircoConfig returns a non-nil pointer", func(t *testing.T) {
		circo := NewCircoConfig()
		if circo == nil {
			t.Fatal("expected non-nil pointer")
		}
	})

	t.Run("beaconsIPAddrsForDomain", func(t *testing.T) {
		circo := &CircoConfig{
			Beacons: map[string]CircoBeaconsDomain{
				"api.ooni.io": {
					IPAddrs: []string{"162.55.247.208"},
					SNIs:    []string{},
				},
			},
			Version: 0,
		}

		t.Run("for known beacon", func(t *testing.T) {
			expect := []string{"162.55.247.208"}
			got := circo.beaconsIPAddrsForDomain("api.ooni.io")
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("for another host", func(t *testing.T) {
			var expect []string
			got := circo.beaconsIPAddrsForDomain("www.example.com")
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("allServerNamesForDomainIncludingDomain", func(t *testing.T) {
		circo := &CircoConfig{
			Beacons: map[string]CircoBeaconsDomain{
				"api.ooni.io": {
					IPAddrs: []string{},
					SNIs:    []string{"x.org"},
				},
			},
			Version: 0,
		}

		t.Run("for known beacon", func(t *testing.T) {
			expect := []string{"api.ooni.io", "x.org"}
			got := circo.allServerNamesForDomainIncludingDomain("api.ooni.io")
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("for another host", func(t *testing.T) {
			expect := []string{"twitter.com"}
			got := circo.allServerNamesForDomainIncludingDomain("twitter.com")
			if diff := cmp.Diff(expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}
