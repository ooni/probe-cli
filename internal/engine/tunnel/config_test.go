package tunnel

import (
	"testing"

	"github.com/apex/log"
)

func TestConfigLoggerDefault(t *testing.T) {
	config := &Config{}
	if config.logger() != defaultLogger {
		t.Fatal("not the logger we expected")
	}
}

func TestConfigLoggerCustom(t *testing.T) {
	config := &Config{Logger: log.Log}
	if config.logger() != log.Log {
		t.Fatal("not the logger we expected")
	}
}
