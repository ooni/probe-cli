//go:build !ooni_psiphon_config

package engine

import (
	"context"
	"errors"
)

// FetchPsiphonConfig fetches psiphon config from the API.
func (s *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	clnt, err := s.newOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	return clnt.FetchPsiphonConfig(ctx)
}

// sessionTunnelEarlySession is the early session that we pass
// to tunnel.Start to fetch the Psiphon configuration.
type sessionTunnelEarlySession struct{}

// errPsiphonNoEmbeddedConfig indicates that there is no
// embedded psiphong config file in this binary.
var errPsiphonNoEmbeddedConfig = errors.New("no embedded configuration file")

// FetchPsiphonConfig implements tunnel.Session.FetchPsiphonConfig.
func (s *sessionTunnelEarlySession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, errPsiphonNoEmbeddedConfig
}

// CheckEmbeddedPsiphonConfig checks whether we can load psiphon's config
func CheckEmbeddedPsiphonConfig() error {
	return nil
}
