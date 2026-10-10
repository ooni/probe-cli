package signalclient

import (
	"testing"
	"time"
)

func TestIdleDurationIsClamped(t *testing.T) {
	for in, want := range map[int64]time.Duration{-5: 0, 0: 0, 30: 30 * time.Second, 60: 60 * time.Second, 90: 60 * time.Second, 1 << 62: 60 * time.Second} {
		if got := idleDuration(in); got != want {
			t.Errorf("idleDuration(%d) = %v, want %v", in, got, want)
		}
	}
}
