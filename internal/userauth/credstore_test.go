package userauth

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

func TestCredStoreRoundTrip(t *testing.T) {
	store := NewCredStore(&kvstore.Memory{})

	// An empty store returns a zero credential.
	if got := store.Get(); got.Credential != "" {
		t.Fatal("expected empty credential from empty store, got", got.Credential)
	}

	cred := StoredCredential{
		Credential:      "base64credential",
		ManifestVersion: "v1",
		PublicParams:    "base64params",
	}
	if err := store.Set(cred); err != nil {
		t.Fatal(err)
	}

	got := store.Get()
	if got != cred {
		t.Fatalf("round-trip mismatch: got %+v want %+v", got, cred)
	}
}
