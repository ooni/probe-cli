package probeservices_test

import (
	"context"
	"testing"
)

func TestGetTestHelpers(t *testing.T) {
	testhelpers, err := newclient().GetTestHelpers(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(testhelpers) <= 1 {
		t.Fatal("no returned test helpers?!")
	}
}
