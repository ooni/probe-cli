//go:build !android && !ios

package tunnel

import (
	"github.com/cretz/bine/tor"
)

func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	// Implementation note: here we make sure that we're not going to
	// execute a binary called "tor" in the current directory on Windows
	// as documented in https://blog.golang.org/path-security.
	exePath, err := config.execabsLookPath(config.torBinary())
	if err != nil {
		return nil, err
	}
	return &tor.StartConf{
		ExePath:   exePath,
		DataDir:   dataDir,
		ExtraArgs: extraArgs,
		NoHush:    true,
	}, nil
}
