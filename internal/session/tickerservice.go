package session

import (
	"context"
	"time"
)

// TickerEvent is an event emmitted by the [tickerService].
type TickerEvent struct {
	// ElapsedTime is the elapsed time since when the
	// long running operation has been started.
	ElapsedTime time.Duration
}

// tickerServer is a ticker that emits a tick while a long running
// operation is alive. We use this ticker in the UI to make progress
// bars increase as well as to unblock the UI code and know when
// an operation has been running for too much time.
type tickerService struct {
	cancel context.CancelFunc
	sess   *Session
}

// newTickerService creates a [tickerService] running in the background
// and using the [Session] to emit [TickerEvents]. You should use the
// stop method of the [tickerService] to stop emitting events.
func newTickerService(ctx context.Context, sess *Session) *tickerService {
	ctx, cancel := context.WithCancel(ctx)
	ts := &tickerService{
		cancel: cancel,
		sess:   sess,
	}
	go ts.mainloop(ctx)
	return ts
}

// mainloop is the main loop of the [tickerService].
func (ts *tickerService) mainloop(ctx context.Context) {
	t0 := time.Now()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			ts.sess.maybeEmit(&Event{
				Ticker: &TickerEvent{
					ElapsedTime: t.Sub(t0),
				},
			})
		case <-ctx.Done():
			return
		}
	}
}

// stop stops the [tickerService] background goroutine.
func (ts *tickerService) stop() {
	ts.cancel()
}
