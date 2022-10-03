package tunnel

//
// torsf: Tor+snowflake tunnel
//

import (
	"context"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

// torsfStart starts the torsf (tor+snowflake) tunnel
func torsfStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	config.logger().Infof("tunnel: starting snowflake with %s rendezvous method", config.snowflakeRendezvousMethod())
	if err := ctx.Err(); err != nil {
		return nil, DebugInfo{}, err
	}

	// 1. start a listener using snowflake
	sfdialer, err := newSnowflakeDialer(config)
	if err != nil {
		return nil, DebugInfo{}, err
	}
	ptl := config.sfNewPTXListener(ctx, sfdialer)
	if err := ptl.Start(); err != nil {
		return nil, DebugInfo{}, err
	}

	// 2. append arguments to the configuration
	extraArguments := []string{
		"UseBridges", "1",
		"ClientTransportPlugin", ptl.AsClientTransportPluginArgument(),
		"Bridge", sfdialer.AsBridgeArgument(),
	}
	config.TorArgs = append(config.TorArgs, extraArguments...)

	// 3. start tor as we would normally do
	torTunnel, debugInfo, err := config.sfTorStart(ctx, config)
	debugInfo.Name = "torsf"
	if err != nil {
		ptl.Stop()
		return nil, debugInfo, err
	}

	// 4. wrap the tunnel and the listener
	tsft := &torsfTunnel{
		torTunnel:  torTunnel,
		sfListener: ptl,
	}
	return tsft, debugInfo, nil
}

func (c *Config) sfNewPTXListener(ctx context.Context, sfdialer *ptx.SnowflakeDialer) (out torsfPTXListener) {
	out = &ptx.Listener{
		ExperimentByteCounter: nil,
		ListenSocks:           c.testSfListenSocks,
		Logger:                c.logger(),
		PTDialer:              sfdialer,
		SessionByteCounter:    bytecounter.ContextSessionByteCounter(ctx),
	}
	if c.testSfWrapPTXListener != nil {
		out = c.testSfWrapPTXListener(out)
	}
	return
}

// torsfPTXListener is an abstract ptx.Listener.
type torsfPTXListener interface {
	Start() error
	Stop()
	AsClientTransportPluginArgument() string
}

func (c *Config) sfTorStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	if c.testSfTorStart != nil {
		return c.testSfTorStart(ctx, config)
	}
	return torStart(ctx, config)
}

// newSnowflakeDialer returns the correct snowflake dialer.
func newSnowflakeDialer(config *Config) (*ptx.SnowflakeDialer, error) {
	rm, err := ptx.NewSnowflakeRendezvousMethod(config.snowflakeRendezvousMethod())
	if err != nil {
		return nil, err
	}
	sfDialer := ptx.NewSnowflakeDialerWithRendezvousMethod(rm)
	return sfDialer, nil
}

// torsfTunnel implements Tunnel
type torsfTunnel struct {
	torTunnel  Tunnel
	sfListener torsfListener
}

// torsfListener is torsfTunnel's view of a ptx listener for snowflake
type torsfListener interface {
	Stop()
}

var _ Tunnel = &torsfTunnel{}

// BootstrapTime implements Tunnel
func (tt *torsfTunnel) BootstrapTime() time.Duration {
	return tt.torTunnel.BootstrapTime()
}

// SOCKS5ProxyURL implements Tunnel
func (tt *torsfTunnel) SOCKS5ProxyURL() *url.URL {
	return tt.torTunnel.SOCKS5ProxyURL()
}

// Stop implements Tunnel
func (tt *torsfTunnel) Stop() {
	tt.torTunnel.Stop()
	tt.sfListener.Stop()
}
