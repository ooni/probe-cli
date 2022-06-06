package netx

//
// Dialer from Config.
//

import (
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewDialer creates a new Dialer from the specified config.
func NewDialer(config Config) model.Dialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	logger := model.ValidLoggerOrDefault(config.Logger)
	d := netxlite.NewDialerWithResolver(
		logger, config.FullResolver, config.Saver.NewConnectObserver(),
		config.ReadWriteSaver.NewReadWriteObserver(),
	)
	d = netxlite.NewMaybeProxyDialer(d, config.ProxyURL)
	d = bytecounter.MaybeWrapWithContextAwareDialer(config.ContextByteCounting, d)
	return d
}
