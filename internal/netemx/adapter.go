package netemx

//
// Code to adapt [netem.UnderlyingNetwork] to [model.UnderlyingNetwork].
//

import (
	"time"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// WithCustomTProxy executes the given function using the given [netem.UnderlyingNetwork]
// as the [model.UnderlyingNetwork] used by the [netxlite] package.
func WithCustomTProxy(tproxy netem.UnderlyingNetwork, function func()) {
	// Implementation note: we use an adapter to reduce timeouts to avoid making the CI run too long
	// which encourages to skip some tests in short mode, which reduces out confidence in the tree
	netxlite.WithCustomTProxy(
		&adapterReduceTimeouts{
			&netxlite.NetemUnderlyingNetworkAdapter{UNet: tproxy},
		},
		function,
	)
}

// adapterReduceTimeouts is a [model.UnderlyingNetwork] that reduces the timeouts
type adapterReduceTimeouts struct {
	model.UnderlyingNetwork
}

// DialTimeout implements [model.UnderlyingNetwork].
func (art *adapterReduceTimeouts) DialTimeout() time.Duration {
	return time.Second
}
