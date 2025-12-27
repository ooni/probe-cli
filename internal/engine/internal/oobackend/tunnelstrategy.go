package oobackend

import (
	"errors"
	"net/http"
)

// tunnelStrategy is a strategy where we use a tunnel.
type tunnelStrategy struct {
	Broker HTTPTunnelBroker // we fail if it's nil
	Name   string           // required
	Info   *strategyInfo    // required
}

// Do implements strategy.Do.
func (s *tunnelStrategy) Do(req *http.Request) (*http.Response, error) {
	resp, err := s.do(req)
	s.Info.updatescore(err) // track the strategy score
	return resp, err
}

// ErrNoBroker indicates that no broker is configured.
var ErrNoBroker = errors.New("oobackend: broker is not configured")

// do gets an HTTPTunnel instance and calls its Do method.
func (s *tunnelStrategy) do(req *http.Request) (*http.Response, error) {
	if s.Broker == nil {
		return nil, ErrNoBroker
	}
	tun, err := s.Broker.New(req.Context(), s.Name)
	if err != nil {
		return nil, err
	}
	return tun.Do(req)
}

// StrategyInfo implements strategy.StrategyInfo.
func (s *tunnelStrategy) StrategyInfo() *strategyInfo {
	return s.Info
}
