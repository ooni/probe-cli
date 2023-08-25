package webconnectivityqa

import (
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewSession(t *testing.T) {
	sess := newSession(http.DefaultClient, model.DiscardLogger)

	t.Run("GetTestHelpers returns Web Connectivity test helpers", func(t *testing.T) {
		// using the empty string such that, when we eventually complicate the
		// mock implementation, we'll get an error in this test
		ths, good := sess.GetTestHelpersByName("")
		if !good {
			t.Fatal("expected good to be true")
		}
		if len(ths) < 1 {
			t.Fatal("expected to see at least a test helper")
		}
	})

	t.Run("we have a default HTTP client", func(t *testing.T) {
		if sess.DefaultHTTPClient() == nil {
			t.Fatal("expected non-nil default HTTP client")
		}
	})

	t.Run("we have a default logger", func(t *testing.T) {
		if sess.Logger() == nil {
			t.Fatal("expected non-nil logger")
		}
	})

	t.Run("LookupASN works as intended", func(t *testing.T) {
		t.Run("for IP addresses inside the 130.192.91.x address space", func(t *testing.T) {
			asn, org, err := sess.LookupASN("130.192.91.155")
			if err != nil {
				t.Fatal(err)
			}
			if asn != 155 {
				t.Fatal("unexpected ASN")
			}
			if org != "Org 155" {
				t.Fatal("unexpected org")
			}
		})

		t.Run("outside of the address space", func(t *testing.T) {
			asn, org, err := sess.LookupASN("10.0.0.1")
			if err == nil {
				t.Fatal("expected an error here")
			}
			if asn != 0 {
				t.Fatal("unexpected ASN")
			}
			if org != "" {
				t.Fatal("unexpected org")
			}
		})
	})

	t.Run("we have an user agent", func(t *testing.T) {
		if sess.UserAgent() == "" {
			t.Fatal("expected non-empty user agent")
		}
	})

	t.Run("we have a resolver IP", func(t *testing.T) {
		if sess.ResolverIP() == "" {
			t.Fatal("expected non-empty resolver IP")
		}
	})
}
