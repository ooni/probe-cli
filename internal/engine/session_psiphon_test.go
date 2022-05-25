//go:build ooni_psiphon_config

package engine

import (
	"context"
	"testing"
)

func TestSessionEmbeddedPsiphonConfig(t *testing.T) {
	s := &Session{}
	data, err := s.FetchPsiphonConfig(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if data == nil {
		t.Fatal("expected non-nil data here")
	}
}

func TestCheckEmbeddedPsiphonConfig(t *testing.T) {
	if err := CheckEmbeddedPsiphonConfig(); err != nil {
		t.Fatal(err)
	}
}
