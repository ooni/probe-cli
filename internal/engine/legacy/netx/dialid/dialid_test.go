package dialid

import (
	"context"
	"testing"
)

func TestGood(t *testing.T) {
	ctx := context.Background()
	id := ContextDialID(ctx)
	if id != 0 {
		t.Fatal("unexpected ID for empty context")
	}
	ctx = WithDialID(ctx)
	id = ContextDialID(ctx)
	if id != 1 {
		t.Fatal("expected ID equal to 1")
	}
	ctx = WithDialID(ctx)
	id = ContextDialID(ctx)
	if id != 2 {
		t.Fatal("expected ID equal to 2")
	}
}
