package ndt7

import "time"

type (
	callbackJSON        func(data []byte) error
	callbackPerformance func(elapsed time.Duration, count int64)
)
