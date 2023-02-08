//go:build !ooni_psiphon_config

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

// SessionTunnelEarlySession is the early session that we pass
// to tunnel.Start to fetch the Psiphon configuration.
type SessionTunnelEarlySession struct{}

// errPsiphonNoEmbeddedConfig indicates that there is no
// embedded psiphong config file in this binary.
var errPsiphonNoEmbeddedConfig = errors.New("no embedded configuration file")

// FetchPsiphonConfig implements tunnel.Session.FetchPsiphonConfig.
func (s *SessionTunnelEarlySession) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	return nil, errPsiphonNoEmbeddedConfig
}

// CheckEmbeddedPsiphonConfig checks whether we can load psiphon's config
func CheckEmbeddedPsiphonConfig() error {
	return nil
}
