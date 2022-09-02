package tunnel

import (
	"context"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/ptx"
)

func torsfStart(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	config.logger().Infof("tunnel: starting snowflake with %s rendezvous method", config.snowflakeRendezvousMethod())

	// 1. start a listener using snowflake
	sfdialer, err := newSnowflakeDialer(config)
	if err != nil {
		return nil, DebugInfo{}, err
	}
	ptl := &ptx.Listener{
		ExperimentByteCounter: nil,
		Logger:                config.logger(),
		PTDialer:              sfdialer,
		SessionByteCounter:    bytecounter.ContextSessionByteCounter(ctx),
	}
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
	torTunnel, debugInfo, err := torStart(ctx, config)
	if err != nil {
		ptl.Stop()
		return nil, debugInfo, err
	}

	torsfTunnel := &torsfTunnel{
		torTunnel:  torTunnel,
		sfListener: ptl,
	}
	return torsfTunnel, debugInfo, nil
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
