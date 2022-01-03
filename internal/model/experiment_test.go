package model

import (
	"testing"

	"github.com/apex/log"
)

func TestPrinterCallbacksCallbacks(t *testing.T) {
	printer := NewPrinterCallbacks(log.Log)
	printer.OnProgress(0.4, "progress")
}
