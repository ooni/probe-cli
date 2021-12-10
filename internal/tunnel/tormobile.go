//go:build ios || android

package tunnel

import (
	"github.com/cretz/bine/tor"
	"github.com/ooni/go-libtor"
)

// getTorStartConf in this configuration uses github.com/ooni/go-libtor.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	config.logger().Info("tunnel: tor: using ooni/go-libtor")
	return &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DataDir:        dataDir,
		ExtraArgs:      extraArgs,
		NoHush:         true,
	}, nil
}
