//go:build !android && !ios && !ooni_libtor

package tunnel

//
// This file implements our strategy for running tor on desktop in most
// configurations except for the ooni_libtor case, where we build tor and
// its dependencies for Linux. The purpuse of this special case it that
// of testing the otherwise untested code that would run on Android.
//

import (
	"strings"

	"github.com/cretz/bine/tor"
)

// getTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	exePath, err := config.torBinary()
	if err != nil {
		config.logger().Warnf("cannot find tor binary: %s", err.Error())
		return nil, err
	}
	config.logger().Infof("tunnel: tor: exec: %s %s %s", exePath,
		dataDir, strings.Join(extraArgs, " "))
	return &tor.StartConf{
		ExePath:   exePath,
		DataDir:   dataDir,
		ExtraArgs: extraArgs,
		NoHush:    true,
	}, nil
}
