//go:build android || ooni_libtor

package tunnel

// This file implements an experimental strategy for running tor.

import (
	"errors"
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/libtor"
)

// getTorStartConf in this configuration returns a tor.StartConf
// configured to run the version of tor we embed as a library.
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
