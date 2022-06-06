package netx

//
// QUICDialer from Config.
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewQUICDialer creates a new QUICDialer using the given Config.
func NewQUICDialer(config Config) model.QUICDialer {
	if config.FullResolver == nil {
		config.FullResolver = NewResolver(config)
	}
	// TODO(bassosimone): we should count the bytes consumed by this QUIC dialer
	ql := config.ReadWriteSaver.WrapQUICListener(netxlite.NewQUICListener())
	logger := model.ValidLoggerOrDefault(config.Logger)
	return netxlite.NewQUICDialerWithResolver(ql, logger, config.FullResolver, config.Saver)
}
