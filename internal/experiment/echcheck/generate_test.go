package echcheck

import (
	"crypto/rand"
	"testing"
)

func TestParseableGREASEConfigList(t *testing.T) {
	// A GREASE extension that can't be parsed is invalid.
	grease, err := generateGreaseyECHConfigList(rand.Reader, "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := parseECHConfigList(grease); err != nil {
		t.Fatal(err)
	}
}
