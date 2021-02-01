package transactionid

import (
	"context"
	"testing"
)

func TestGood(t *testing.T) {
	ctx := context.Background()
	id := ContextTransactionID(ctx)
	if id != 0 {
		t.Fatal("unexpected ID for empty context")
	}
	ctx = WithTransactionID(ctx)
	id = ContextTransactionID(ctx)
	if id != 1 {
		t.Fatal("expected ID equal to 1")
	}
	ctx = WithTransactionID(ctx)
	id = ContextTransactionID(ctx)
	if id != 2 {
		t.Fatal("expected ID equal to 2")
	}
}
