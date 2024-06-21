package enginenetx

import (
	"context"
	"testing"
)

func TestBridgesPolicyV2(t *testing.T) {
	t.Run("for domains for which we don't have bridges", func(t *testing.T) {
		p := &bridgesPolicyV2{}

		tactics := p.LookupTactics(context.Background(), "www.example.com", "443")

		var count int
		for range tactics {
			count++
		}

		if count != 0 {
			t.Fatal("expected to see zero tactics")
		}
	})

	t.Run("for the api.ooni.io domain", func(t *testing.T) {
		p := &bridgesPolicyV2{}

		tactics := p.LookupTactics(context.Background(), "api.ooni.io", "443")

		var count int
		for tactic := range tactics {
			count++

			// for each generated tactic, make sure we're getting the
			// expected value for each of the fields

			if tactic.Port != "443" {
				t.Fatal("the port should always be 443")
			}

			if tactic.Address != "162.55.247.208" {
				t.Fatal("the host should always be 162.55.247.208")
			}

			if tactic.InitialDelay != 0 {
				t.Fatal("unexpected InitialDelay")
			}

			if tactic.SNI == "api.ooni.io" {
				t.Fatal("we should not see the `api.ooni.io` SNI on the wire")
			}

			if tactic.VerifyHostname != "api.ooni.io" {
				t.Fatal("the VerifyHostname field should always be like `api.ooni.io`")
			}
		}

		if count <= 0 {
			t.Fatal("expected to see at least one tactic")
		}
	})
}
