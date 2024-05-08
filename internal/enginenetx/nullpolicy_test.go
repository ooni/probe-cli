package enginenetx

import (
	"context"
	"testing"
)

func TestNullPolicy(t *testing.T) {
	p := &nullPolicy{}
	var count int
	for range p.LookupTactics(context.Background(), "api.ooni.io", "443") {
		count++
	}
	if count != 0 {
		t.Fatal("should have not returned any policy")
	}
}
