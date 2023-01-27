//go:build ios || (android && !ooni_libtor)

package tunnel

//
// This file implements our old strategy for running tor on mobile, which
// is based on integrating github.com/ooni/go-libtor. We currently only use
// this stategy on iOS. See https://github.com/ooni/probe/issues/2365.
//

import (
	"strings"

	"github.com/cretz/bine/tor"
	"github.com/ooni/go-libtor"
)

// getTorStartConf in this configuration uses github.com/ooni/go-libtor.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	config.logger().Infof("tunnel: tor: exec: <ooni/go-libtor> %s %s",
		dataDir, strings.Join(extraArgs, " "))
	return &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DataDir:        dataDir,
		ExtraArgs:      extraArgs,
		NoHush:         true,
	}, nil
}
