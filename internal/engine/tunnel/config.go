package tunnel

import (
	"context"
	"os"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
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

	// testTorStart allows us to mock tor.Start.
	testTorStart func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error)

	// testTorEnableNetwork allows us to fake a failure when
	// telling to the tor daemon to enable the network.
	testTorEnableNetwork func(ctx context.Context, tor *tor.Tor, wait bool) error

	// testTorGetInfo allows us to fake a failure when
	// getting info from the tor control port.
	testTorGetInfo func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error)
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

// torStart calls either testTorStart or tor.Start.
func (c *Config) torStart(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
	if c.testTorStart != nil {
		return c.testTorStart(ctx, conf)
	}
	return tor.Start(ctx, conf)
}

// torEnableNetwork calls either testTorEnableNetwork or tor.EnableNetwork.
func (c *Config) torEnableNetwork(ctx context.Context, tor *tor.Tor, wait bool) error {
	if c.testTorEnableNetwork != nil {
		return c.testTorEnableNetwork(ctx, tor, wait)
	}
	return tor.EnableNetwork(ctx, wait)
}

// torGetInfo calls either testTorGetInfo or ctrl.GetInfo.
func (c *Config) torGetInfo(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error) {
	if c.testTorGetInfo != nil {
		return c.testTorGetInfo(ctrl, keys...)
	}
	return ctrl.GetInfo(keys...)
}
