//go:build ooni_libtor && android

package tunnel

//
// This file implements the ooni_libtor strategy of embedding tor. We manually
// compile tor and its dependencies and link against it. We currently only adopt
// this technique for Android. We may possibly migrate also iOS in the future,
// provided that this functionality proves to be stable in the 3.17 cycle.
//
// See https://github.com/ooni/probe/issues/2365.
//

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
