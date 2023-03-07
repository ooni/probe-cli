package main

import (
	"context"
	"time"
)

type tickerResponse struct {
	ElapsedTime float64 `json:",omitempty"`
}

// runTicker emits a ticker event every second. It is a subtask
// associated with all tasks.
func runTicker(ctx context.Context, close chan any,
	emitter taskMaybeEmitter, req *request, start time.Time) {
	var resp *response
	ticker := time.NewTicker(time.Second)
	for {
		select {
		case <-close:
			return
		case <-ticker.C:
			resp.Ticker.ElapsedTime = time.Since(start).Seconds()
			emitter.maybeEmitEvent(resp)
		}
	}
}
