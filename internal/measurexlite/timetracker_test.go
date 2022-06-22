package measurexlite

import (
	"testing"
	"time"
)

func TestTimeTracker(t *testing.T) {
	t.Run("Since", func(t *testing.T) {
		t.Run("with nil TimeTracker", func(t *testing.T) {
			zeroTime := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
			var tt *timeTracker
			duration := tt.Since(zeroTime)
			if duration <= 0 {
				t.Fatal("unexpected duration")
			}
		})

		t.Run("with nonnil TimeTracker", func(t *testing.T) {
			zeroTime := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
			tt := &timeTracker{}
			duration := tt.Since(zeroTime)
			if duration != time.Second {
				t.Fatal("unexpected duration")
			}
		})
	})

	t.Run("Sub", func(t *testing.T) {
		t.Run("with nil TimeTracker", func(t *testing.T) {
			t0 := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
			t1 := time.Now()
			var tt *timeTracker
			duration := tt.Sub(t1, t0)
			if duration <= 0 {
				t.Fatal("unexpected duration")
			}
		})

		t.Run("with nonnil TimeTracker", func(t *testing.T) {
			t0 := time.Date(2022, 01, 01, 00, 00, 00, 00, time.UTC)
			t1 := time.Now()
			tt := &timeTracker{}
			duration := tt.Sub(t1, t0)
			if duration != time.Second {
				t.Fatal("unexpected duration")
			}
		})
	})
}
