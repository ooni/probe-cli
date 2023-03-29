//go:build !android && !ios && !ooni_libtor

package tortunnel

//
// This file implements our strategy for running tor on desktop in most
// configurations except for the ooni_libtor case, where we build tor and
// its dependencies for Linux. The purpuse of this special case it that
// of testing the otherwise untested code that would run on Android.
//

import (
	"github.com/cretz/bine/tor"
	"golang.org/x/sys/execabs"
)

// newTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func newTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	// determine the logget to use.
	logger := config.logger()

	// determine the tor binary to use.
	torBinary := config.TorBinary
	if torBinary == "" {
		var err error
		torBinary, err = execabs.LookPath("tor")
		if err != nil {
			return nil, err
		}
	}
	logger.Infof("tortunnel: using this tor binary: %s", torBinary)

	// generate and return the config.
	tsc := &tor.StartConf{
		ExePath:   torBinary,
		DataDir:   dataDir,
		ExtraArgs: extraArgs,
	}
	return tsc, nil
}
