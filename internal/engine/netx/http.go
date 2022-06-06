package netx

//
// HTTPTransport from Config.
//

import (
	"crypto/tls"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewHTTPTransport creates a new HTTPRoundTripper from the given Config.
func NewHTTPTransport(config Config) model.HTTPTransport {
	if config.Dialer == nil {
		// TODO(https://github.com/ooni/probe/issues/2121#issuecomment-1147424810)
		config.Dialer = NewDialer(config)
	}
	if config.TLSDialer == nil {
		// TODO(https://github.com/ooni/probe/issues/2121#issuecomment-1147424810)
		config.TLSDialer = NewTLSDialer(config)
	}
	if config.QUICDialer == nil {
		// TODO(https://github.com/ooni/probe/issues/2121#issuecomment-1147424810)
		config.QUICDialer = NewQUICDialer(config)
	}
	tInfo := allTransportsInfo[config.HTTP3Enabled]
	txp := tInfo.Factory(httpTransportConfig{
		Dialer:     config.Dialer,
		Logger:     model.ValidLoggerOrDefault(config.Logger),
		QUICDialer: config.QUICDialer,
		TLSDialer:  config.TLSDialer,
		TLSConfig:  config.TLSConfig,
	})
	// TODO(https://github.com/ooni/probe/issues/2121#issuecomment-1147424810): I am
	// not super convinced by this code because it
	// seems we're currently counting bytes twice in some cases. I think we
	// should review how we're counting bytes and using netx currently.
	txp = config.ByteCounter.MaybeWrapHTTPTransport(txp)                 // WAI with ByteCounter == nil
	const defaultSnapshotSize = 0                                        // means: use the default snapsize
	return config.Saver.MaybeWrapHTTPTransport(txp, defaultSnapshotSize) // WAI with Saver == nil
}

// httpTransportInfo contains the constructing function as well as the transport name
type httpTransportInfo struct {
	Factory       func(httpTransportConfig) model.HTTPTransport
	TransportName string
}

var allTransportsInfo = map[bool]httpTransportInfo{
	false: {
		Factory:       newHTTPTransport,
		TransportName: "tcp",
	},
	true: {
		Factory:       newHTTP3Transport,
		TransportName: "quic",
	},
}

// httpTransportConfig contains configuration for constructing an HTTPTransport.
//
// All the fields in this structure MUST be initialized.
type httpTransportConfig struct {
	Dialer     model.Dialer
	Logger     model.Logger
	QUICDialer model.QUICDialer
	TLSDialer  model.TLSDialer
	TLSConfig  *tls.Config
}

// newHTTP3Transport creates a new HTTP3Transport instance.
func newHTTP3Transport(config httpTransportConfig) model.HTTPTransport {
	return netxlite.NewHTTP3Transport(config.Logger, config.QUICDialer, config.TLSConfig)
}

// newHTTPTransport creates a new "system" HTTP transport.
func newHTTPTransport(config httpTransportConfig) model.HTTPTransport {
	return netxlite.NewHTTPTransport(config.Logger, config.Dialer, config.TLSDialer)
}
