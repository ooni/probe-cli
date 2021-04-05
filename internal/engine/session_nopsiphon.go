// +build !ooni_psiphon_config

package engine

import (
	"context"
	"errors"
)

// FetchPsiphonConfig fetches psiphon config from the API.
func (s *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	clnt, err := s.NewOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	return clnt.FetchPsiphonConfig(ctx)
}

// sessionTunnelEarlySession is the early session that we pass
// to tunnel.Start to fetch the Psiphon configuration.
type sessionTunnelEarlySession struct{}

// FetchPsiphonConfig implements tunnel.Session.FetchPsiphonConfig.
func (s *sessionTunnelEarlySession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, errors.New("no embedded configuration file")
}
