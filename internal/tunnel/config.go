package tunnel

import (
	"context"
	"errors"
	"net"
	"os"

	"github.com/armon/go-socks5"
	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/execabs"
)

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

	// SnowflakeRendezvous is the OPTIONAL rendezvous
	// method for snowflake
	SnowflakeRendezvous string

	// TunnelDir is the MANDATORY directory in which the tunnel SHOULD
	// store its state, if any. If this field is empty, the
	// Start function fails with ErrEmptyTunnelDir.
	TunnelDir string

	// Logger is the optional logger to use. If empty we use a default
	// implementation that does not emit any output.
	Logger model.Logger

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

	// testTorStart allows us to mock tor.Start.
	testTorStart func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error)

	// testTorProtocolInfo allows us to mock getting protocol info.
	testTorProtocolInfo func(tor *tor.Tor) (*control.ProtocolInfo, error)

	// testTorEnableNetwork allows us to fake a failure when
	// telling to the tor daemon to enable the network.
	testTorEnableNetwork func(ctx context.Context, tor *tor.Tor, wait bool) error

	// testTorGetInfo allows us to fake a failure when
	// getting info from the tor control port.
	testTorGetInfo func(ctrl *control.Conn, keys ...string) ([]*control.KeyVal, error)
}

// snowflakeRendezvousMethod returns the rendezvous method that snowflake should use
func (c *Config) snowflakeRendezvousMethod() string {
	if c.SnowflakeRendezvous != "" {
		return c.SnowflakeRendezvous
	}
	return "domain_fronting"
}

// logger returns the logger to use.
func (c *Config) logger() model.Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return model.DiscardLogger
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

// ooniTorBinaryEnv is the name of the environment variable
// we're using to get the path to the tor binary when we are
// being run by the ooni/probe-desktop application.
const ooniTorBinaryEnv = "OONI_TOR_BINARY"

// torBinary returns the tor binary path.
//
// Here's is the algorithm:
//
// 1. if c.TorBinary is set, we use its value;
//
// 2. if os.Getenv("OONI_TOR_BINARY") is set, we use its value;
//
// 3. otherwise, we return "tor".
//
// Implementation note: in cases 1 and 3 we use execabs.LookPath
// to guarantee we're not going to execute a binary outside of the
// PATH (see https://blog.golang.org/path-security for more info
// on how this bug could affect Windows). In case 2, we're instead
// just going to trust the binary set by the probe-desktop app.
func (c *Config) torBinary() (string, error) {
	if c.TorBinary != "" {
		return c.execabsLookPath(c.TorBinary)
	}
	if binary := os.Getenv(ooniTorBinaryEnv); binary != "" {
		return binary, nil
	}
	return c.execabsLookPath("tor")
}

// torStart calls either testTorStart or tor.Start.
func (c *Config) torStart(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error) {
	if c.testTorStart != nil {
		return c.testTorStart(ctx, conf)
	}
	return tor.Start(ctx, conf)
}

// errNoTorControl indicate you passed us a tor with a nil control field.
var errNoTorControl = errors.New("tunnel: no tor control")

// torProtocolInfo calls either testTorProtocolInfo or the
// proper function to get back protocol information.
func (c *Config) torProtocolInfo(tor *tor.Tor) (*control.ProtocolInfo, error) {
	if c.testTorProtocolInfo != nil {
		return c.testTorProtocolInfo(tor)
	}
	if tor.Control == nil {
		return nil, errNoTorControl
	}
	return tor.Control.ProtocolInfo()
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
