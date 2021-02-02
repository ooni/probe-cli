package ndt7

import "time"

func defaultCallbackJSON(data []byte) error {
	return nil
}

func defaultCallbackPerformance(elapsed time.Duration, count int64) {
}
