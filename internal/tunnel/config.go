package tunnel

import (
	"context"
	"net"
	"os"

	"github.com/armon/go-socks5"
	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
	"golang.org/x/sys/execabs"
)

// Logger is the logger to use. Its signature is compatibile
// with the apex/log logger signature.
type Logger interface {
	// Infof formats and emits an informative message
	Infof(format string, v ...interface{})
}

// Config contains the configuration for creating a Tunnel instance. You need
// to fill all the mandatory fields. You SHOULD NOT modify the content of this
// structure while in use, because that may lead to data races.
type Config struct {
	// Name is the MANDATORY name of the tunnel. We support
	// "tor", "psiphon", and "fake" tunnels. You SHOULD
	// use "fake" tunnels only for testing: they don't provide
	// any real tunneling, just a socks5 proxy.
	Name string

	// Session is the MANDATORY measurement session, or a suitable
	// mock of the required functionality. That is, the possibility
	// of obtaining a valid psiphon configuration.
	Session Session

	// TunnelDir is the MANDATORY directory in which the tunnel SHOULD
	// store its state, if any. If this field is empty, the
	// Start function fails with ErrEmptyTunnelDir.
	TunnelDir string

	// Logger is the optional logger to use. If empty we use a default
	// implementation that does not emit any output.
	Logger Logger

	// TorArgs contains the optional arguments that you want us to pass
	// to the tor binary when invoking it. By default we do not
	// pass any extra argument. This flag might be useful to
	// configure pluggable transports.
	TorArgs []string

	// TorBinary is the optional path of the TorBinary we SHOULD be
	// executing. When not set, we execute `tor`.
	TorBinary string

	// testExecabsLookPath allows us to mock exeabs.LookPath
	testExecabsLookPath func(name string) (string, error)

	// testMkdirAll allows us to mock os.MkdirAll in testing code.
	testMkdirAll func(path string, perm os.FileMode) error

	// testNetListen allows us to mock net.Listen in testing code.
	testNetListen func(network string, address string) (net.Listener, error)

	// testSocks5New allows us to mock socks5.New in testing code.
	testSocks5New func(conf *socks5.Config) (*socks5.Server, error)

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

// silentLogger is a logger that does not emit output.
type silentLogger struct{}

// Infof implements Logger.Infof.
func (sl *silentLogger) Infof(format string, v ...interface{}) {}

// defaultLogger is the default logger.
var defaultLogger = &silentLogger{}

// logger returns the logger to use.
func (c *Config) logger() Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return defaultLogger
}

// execabsLookPath calls either testExeabsLookPath or execabs.LookPath
func (c *Config) execabsLookPath(name string) (string, error) {
	if c.testExecabsLookPath != nil {
		return c.testExecabsLookPath(name)
	}
	return execabs.LookPath(name)
}

// mkdirAll calls either testMkdirAll or os.MkdirAll.
func (c *Config) mkdirAll(path string, perm os.FileMode) error {
	if c.testMkdirAll != nil {
		return c.testMkdirAll(path, perm)
	}
	return os.MkdirAll(path, perm)
}

// netListen calls either testNetListen or net.Listen.
func (c *Config) netListen(network string, address string) (net.Listener, error) {
	if c.testNetListen != nil {
		return c.testNetListen(network, address)
	}
	return net.Listen(network, address)
}

// socks5New calls either testSocks5New or socks5.New
func (c *Config) socks5New(conf *socks5.Config) (*socks5.Server, error) {
	if c.testSocks5New != nil {
		return c.testSocks5New(conf)
	}
	return socks5.New(conf)
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

// torBinary returns the tor binary path, if configured, or
// the default path, otherwise.
func (c *Config) torBinary() string {
	if c.TorBinary != "" {
		return c.TorBinary
	}
	return "tor"
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
