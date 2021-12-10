package tunnel

import (
	"os"
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

func TestTorBinaryEnvironmentVariable(t *testing.T) {
	path := "/usr/local/bin/tor"
	os.Setenv(ooniTorBinaryEnv, path)
	config := &Config{}
	res := config.torBinary()
	os.Unsetenv(ooniTorBinaryEnv)
	if res != path {
		t.Fatal("not the result we expected")
	}
}

func TestTorBinarySetTakesPrecedenceOverTheEnvVariable(t *testing.T) {
	path := "/usr/local/bin/tor"
	os.Setenv(ooniTorBinaryEnv, "/usr/local/bin/tor-real")
	config := &Config{TorBinary: path}
	res := config.torBinary()
	os.Unsetenv(ooniTorBinaryEnv)
	if res != path {
		t.Fatal("not the result we expected")
	}
}
