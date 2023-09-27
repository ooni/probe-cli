package mocks

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	utls "gitlab.com/yawning/utls.git"
)

// MeasuringNetwork allows mocking [model.MeasuringNetwork].
type MeasuringNetwork struct {
	MockNewDialerWithoutResolver func(dl model.DebugLogger, w ...model.DialerWrapper) model.Dialer

	MockNewParallelDNSOverHTTPSResolver func(logger model.DebugLogger, URL string) model.Resolver

	MockNewParallelUDPResolver func(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver

	MockNewQUICDialerWithoutResolver func(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer

	MockNewStdlibResolver func(logger model.DebugLogger) model.Resolver

	MockNewTLSHandshakerStdlib func(logger model.DebugLogger) model.TLSHandshaker

	MockNewTLSHandshakerUTLS func(logger model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker

	MockNewUDPListener func() model.UDPListener
}

var _ model.MeasuringNetwork = &MeasuringNetwork{}

// NewDialerWithoutResolver implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewDialerWithoutResolver(dl model.DebugLogger, w ...model.DialerWrapper) model.Dialer {
	return mn.MockNewDialerWithoutResolver(dl, w...)
}

// NewParallelDNSOverHTTPSResolver implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewParallelDNSOverHTTPSResolver(logger model.DebugLogger, URL string) model.Resolver {
	return mn.MockNewParallelDNSOverHTTPSResolver(logger, URL)
}

// NewParallelUDPResolver implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver {
	return mn.MockNewParallelUDPResolver(logger, dialer, address)
}

// NewQUICDialerWithoutResolver implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewQUICDialerWithoutResolver(listener model.UDPListener, logger model.DebugLogger, w ...model.QUICDialerWrapper) model.QUICDialer {
	return mn.MockNewQUICDialerWithoutResolver(listener, logger, w...)
}

// NewStdlibResolver implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewStdlibResolver(logger model.DebugLogger) model.Resolver {
	return mn.MockNewStdlibResolver(logger)
}

// NewTLSHandshakerStdlib implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewTLSHandshakerStdlib(logger model.DebugLogger) model.TLSHandshaker {
	return mn.MockNewTLSHandshakerStdlib(logger)
}

// NewTLSHandshakerUTLS implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewTLSHandshakerUTLS(logger model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
	return mn.MockNewTLSHandshakerUTLS(logger, id)
}

// NewUDPListener implements model.MeasuringNetwork.
func (mn *MeasuringNetwork) NewUDPListener() model.UDPListener {
	return mn.MockNewUDPListener()
}
