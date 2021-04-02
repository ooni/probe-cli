// +build !ooni_psiphon_config

package engine

import "context"

// FetchPsiphonConfig fetches psiphon config from the API.
func (s *Session) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	clnt, err := s.NewOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	return clnt.FetchPsiphonConfig(ctx)
}
