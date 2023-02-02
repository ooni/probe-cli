package probeservices

import (
	"context"
	"testing"
)

func TestGetTestHelpers(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	testhelpers, err := newclient().GetTestHelpers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(testhelpers) <= 1 {
		t.Fatal("no returned test helpers?!")
	}
}
