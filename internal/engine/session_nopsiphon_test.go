//go:build !ooni_psiphon_config
// +build !ooni_psiphon_config

package engine

import (
	"context"
	"errors"
	"testing"
)

func TestEarlySessionNoPsiphonFetchPsiphonConfig(t *testing.T) {
	s := &sessionTunnelEarlySession{}
	out, err := s.FetchPsiphonConfig(context.Background())
	if !errors.Is(err, errPsiphonNoEmbeddedConfig) {
		t.Fatal("not the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil here")
	}
}
