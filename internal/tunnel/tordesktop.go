//go:build !android && !ios

package tunnel

// This file implements our strategy for running tor on desktop.

import "github.com/cretz/bine/tor"

// getTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	exePath, err := config.torBinary()
	if err != nil {
		return nil, err
	}
	config.logger().Infof("tunnel: tor: exec binary: %s", exePath)
	return &tor.StartConf{
		ExePath:   exePath,
		DataDir:   dataDir,
		ExtraArgs: extraArgs,
		NoHush:    true,
	}, nil
}
