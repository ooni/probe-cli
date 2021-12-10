//go:build ios || android

package tunnel

import (
	"github.com/cretz/bine/tor"
	"github.com/ooni/go-libtor"
)

func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	return &tor.StartConf{
		ProcessCreator: libtor.Creator,
		DataDir:        dataDir,
		ExtraArgs:      extraArgs,
		NoHush:         true,
	}, nil
}
