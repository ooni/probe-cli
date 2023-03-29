package tortunnel

import (
	"context"

	"github.com/cretz/bine/control"
	"github.com/cretz/bine/tor"
)

// Dependencies contains dependencies allowing to mock [Start] for testing.
type Dependencies struct {
	// Start should be equivalent to [tor.Start].
	Start func(ctx context.Context, conf *tor.StartConf) (*tor.Tor, error)

	// TorControlProtocolInfo should be equivalent to calling the
	// ProtocolInfo method of instance.Control.
	TorControlProtocolInfo func(instance *tor.Tor) (*control.ProtocolInfo, error)

	// TorControlGetInfo should be equivalent to calling the GetInfo
	// method of the instance.Control.
	TorControlGetInfo func(instance *tor.Tor, keys ...string) ([]*control.KeyVal, error)

	// TorEnableNetwork should be equivalent to calling the
	// EnableNetwork method of instance.
	TorEnableNetwork func(ctx context.Context, instance *tor.Tor, wait bool) error
}

var defaultDependencies = &Dependencies{
	Start: tor.Start,
	TorControlProtocolInfo: func(instance *tor.Tor) (*control.ProtocolInfo, error) {
		return instance.Control.ProtocolInfo()
	},
	TorControlGetInfo: func(instance *tor.Tor, keys ...string) ([]*control.KeyVal, error) {
		return instance.Control.GetInfo(keys...)
	},
	TorEnableNetwork: func(ctx context.Context, instance *tor.Tor, wait bool) error {
		return instance.EnableNetwork(ctx, wait)
	},
}
