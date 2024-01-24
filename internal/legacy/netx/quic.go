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
	// TODO(https://github.com/ooni/probe/issues/2121#issuecomment-1147424810): we
	// should count the bytes consumed by this QUIC dialer
	netx := &netxlite.Netx{}
	ql := config.ReadWriteSaver.WrapUDPListener(netxlite.NewUDPListener())
	logger := model.ValidLoggerOrDefault(config.Logger)
	return netx.NewQUICDialerWithResolver(ql, logger, config.FullResolver, config.Saver)
}
