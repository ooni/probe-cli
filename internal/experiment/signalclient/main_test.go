package signalclient

import (
	"os"
	"testing"
	"time"
)

// TestMain shortens the controls' handshake timeout: in the simulated network, most static
// addresses have no host, and a SYN to them is dropped rather than refused.
func TestMain(m *testing.M) {
	controlHandshakeTimeout = 2 * time.Second
	os.Exit(m.Run())
}
