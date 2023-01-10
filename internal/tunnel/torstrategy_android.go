//go:build android

package tunnel

// This file implements our strategy for running tor on android.

import (
	"errors"
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/libtor"
)

// getTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	creator, good := libtor.MaybeCreator()
	if !good {
		return nil, errors.New("no embedded tor")
	}
	config.logger().Infof("tunnel: tor: exec: <internal/libtor> %s %s",
		dataDir, strings.Join(extraArgs, " "))
	return &tor.StartConf{
		ProcessCreator: creator,
		DataDir:        dataDir,
		ExtraArgs:      extraArgs,
		NoHush:         true,
	}, nil
}
