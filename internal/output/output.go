package output

import (
	"github.com/apex/log"
)

// Progress logs a progress type event
func Progress(key string, perc float64, msg string) {
	log.WithFields(log.Fields{
		"type":       "progress",
		"key":        key,
		"percentage": perc,
	}).Info(msg)
}
