//go:build !android && !ios

package tunnel

import (
	"github.com/cretz/bine/tor"
)

// getTorStartConf in this configuration uses torExePath to get a
// suitable tor binary and then executes it.
func getTorStartConf(config *Config, dataDir string, extraArgs []string) (*tor.StartConf, error) {
	// Implementation note: here we make sure that we're not going to
	// execute a binary called "tor" in the current directory on Windows
	// as documented in https://blog.golang.org/path-security.
	//
	// To this end, we make an indirect call to execabs.LookPath.
	exePath, err := config.execabsLookPath(config.torBinary())
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
