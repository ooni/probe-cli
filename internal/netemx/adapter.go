package netemx

//
// Code to adapt [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
//

import (
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WithCustomTProxy executes the given function using the given [netem.UnderlyingNetwork]
// as the [model.UnderlyingNetwork] used by the [netxlite] package.
func WithCustomTProxy(tproxy netem.UnderlyingNetwork, function func()) {
	netxlite.WithCustomTProxy(&netxlite.NetemUnderlyingNetworkAdapter{UNet: tproxy}, function)
}
