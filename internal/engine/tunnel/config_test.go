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

func TestTorBinaryNotSet(t *testing.T) {
	config := &Config{}
	if config.torBinary() != "tor" {
		t.Fatal("not the result we expected")
	}
}

func TestTorBinarySet(t *testing.T) {
	path := "/usr/local/bin/tor"
	config := &Config{TorBinary: path}
	if config.torBinary() != path {
		t.Fatal("not the result we expected")
	}
}
