package portfiltering

import (
	"testing"
	"time"
)

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.delay() != 100*time.Millisecond {
		t.Fatal("invalid default delay")
	}
}
