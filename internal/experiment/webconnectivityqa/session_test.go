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
