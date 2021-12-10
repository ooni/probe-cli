package tunnel

import (
	"errors"
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

func TestConfigTorBinary(t *testing.T) {
	// newConfig is a factory for creating a new config
	//
	// Arguments:
	//
	// - torBinaryPath is the possibly-empty config.TorBinary to use;
	//
	// - realBinaryPath is the possibly-empty path execabs.LookupPath should return;
	//
	// - err is the possbly-nil error that execabs.LookupPath should return.
	//
	// Returns a new *Config.
	newConfig := func(binaryPath, realBinaryPath string, err error) *Config {
		return &Config{
			TorBinary: binaryPath,
			testExecabsLookPath: func(name string) (string, error) {
				if err != nil {
					return "", err
				}
				return realBinaryPath, nil
			},
		}
	}

	// verifyExpectations ensures that config.torBinary() produces in
	// output the expectPath and expectErr result.
	verifyExpectations := func(
		t *testing.T, config *Config, expectPath string, expectErr error) {
		path, err := config.torBinary()
		if !errors.Is(err, expectErr) {
			t.Fatal("not the error we expected", err)
		}
		if path != expectPath {
			t.Fatal("not the path we expected", path)
		}
	}

	t.Run("with empty TorBinary and no tor in PATH", func(t *testing.T) {
		expected := errors.New("no such binary in PATH")
		config := newConfig("", "", expected)
		verifyExpectations(t, config, "", expected)
	})

	t.Run("with empty TorBinary and tor in PATH", func(t *testing.T) {
		expected := "/usr/bin/tor"
		config := newConfig("", expected, nil)
		verifyExpectations(t, config, expected, nil)
	})

	t.Run("with TorBinary and no such binary in PATH", func(t *testing.T) {
		expected := errors.New("no such binary in PATH")
		config := newConfig("tor-real", "", expected)
		verifyExpectations(t, config, "", expected)
	})

	t.Run("with TorBinary and the binary is in PATH", func(t *testing.T) {
		expected := "/usr/bin/tor-real"
		config := newConfig("tor-real", expected, nil)
		verifyExpectations(t, config, expected, nil)
	})

	t.Run("with OONI_TOR_BINARY and empty TorBinary", func(t *testing.T) {
		expected := "./tor.exe"
		os.Setenv(ooniTorBinaryEnv, expected)
		config := newConfig("", expected, errors.New("should not be seen"))
		verifyExpectations(t, config, expected, nil)
	})

	t.Run("with OONI_TOR_BINARY and TorBinary not in PATH", func(t *testing.T) {
		expected := errors.New("no such binary in PATH")
		os.Setenv(ooniTorBinaryEnv, "./tor.exe")
		config := newConfig("tor-real", "", expected)
		verifyExpectations(t, config, "", expected)
	})

	t.Run("with OONI_TOR_BINARY and TorBinary in PATH", func(t *testing.T) {
		expected := "/usr/bin/tor-real"
		os.Setenv(ooniTorBinaryEnv, "./tor.exe")
		config := newConfig("tor-real", expected, nil)
		verifyExpectations(t, config, expected, nil)
	})
}
