package oonimkall

import (
	"context"
	"math"
	"time"
)

const maxTimeout = int64(time.Duration(math.MaxInt64) / time.Second)

func clampTimeout(timeout, max int64) int64 {
	if timeout > max {
		timeout = max
	}
	return timeout
}

func newContext(timeout int64) (context.Context, context.CancelFunc) {
	return newContextEx(timeout, maxTimeout)
}

func newContextEx(timeout, max int64) (context.Context, context.CancelFunc) {
	if timeout > 0 {
		timeout = clampTimeout(timeout, max)
		return context.WithTimeout(
			context.Background(), time.Duration(timeout)*time.Second)
	}
	return context.WithCancel(context.Background())
}
