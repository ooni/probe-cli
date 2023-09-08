package testingx

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNewDNSRoundTripSimulateTimeout(t *testing.T) {
	t.Run("when the context has already been cancelled", func(t *testing.T) {
		rtx := NewDNSRoundTripperSimulateTimeout(time.Second, errors.New("mocked error"))
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // immediately cancel
		resp, err := rtx.RoundTrip(ctx, make([]byte, 128))
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
		if len(resp) != 0 {
			t.Fatal("expected zero-byte resp")
		}
	})
}
