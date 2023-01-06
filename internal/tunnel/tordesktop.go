//go:build !android && !ios

package tunnel

// This file implements our strategy for running tor on desktop.

import (
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/libtor"
)

// getTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	creator, good := libtor.MaybeCreator()
	if good {
		config.logger().Infof("tunnel: tor: exec: <internal/libtor> %s %s",
			dataDir, strings.Join(extraArgs, " "))
		return &tor.StartConf{
			ProcessCreator: creator,
			DataDir:        dataDir,
			ExtraArgs:      extraArgs,
			NoHush:         true,
		}, nil
	}
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
