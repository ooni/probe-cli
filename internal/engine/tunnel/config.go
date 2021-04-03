package tunnel

import (
	"context"
	"os"

	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// Config contains the configuration for creating a Tunnel instance.
type Config struct {
	// Name is the mandatory name of the tunnel. We support
	// "tor" and "psiphon" tunnels.
	Name string

	// Session is the current measurement session.
	Session Session

	// WorkDir is the directory in which the tunnel SHOULD
	// store its state, if any.
	WorkDir string

	// testMkdirAll allows us to mock os.MkdirAll in testing code.
	testMkdirAll func(path string, perm os.FileMode) error

	// testRemoveAll allows us to mock os.RemoveAll in testing code.
	testRemoveAll func(path string) error

	// testStartPsiphon allows us to mock psiphon's clientlib.StartTunnel.
	testStartPsiphon func(ctx context.Context, config []byte,
		workdir string) (*clientlib.PsiphonTunnel, error)
}

// mkdirAll calls either testMkdirAll or os.MkdirAll.
func (c *Config) mkdirAll(path string, perm os.FileMode) error {
	if c.testMkdirAll != nil {
		return c.testMkdirAll(path, perm)
	}
	return os.MkdirAll(path, perm)
}

// removeAll calls either testRemoveAll or os.RemoveAll.
func (c *Config) removeAll(path string) error {
	if c.testRemoveAll != nil {
		return c.testRemoveAll(path)
	}
	return os.RemoveAll(path)
}

// startPsiphon calls either testStartPsiphon or psiphon's clientlib.StartTunnel.
func (c *Config) startPsiphon(ctx context.Context, config []byte,
	workdir string) (*clientlib.PsiphonTunnel, error) {
	if c.testStartPsiphon != nil {
		return c.testStartPsiphon(ctx, config, workdir)
	}
	return clientlib.StartTunnel(ctx, config, "", clientlib.Parameters{
		DataRootDirectory: &workdir}, nil, nil)
}
