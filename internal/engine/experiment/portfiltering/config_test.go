package portfiltering

import (
	"testing"
)

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.testhelper() != "http://127.0.0.1" {
		t.Fatal("invalid default testhelper")
	}
}
