package main

//
// Code to create an *engine.Session.
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// parseProxyURL returns the proper proxy URL or nil if it's not configured.
func parseProxyURL(proxyURL string) (*url.URL, error) {
	if proxyURL == "" {
		return nil, nil
	}
	return url.Parse(proxyURL)
}

// newSession creates a new *engine.Session from the given Config.
func newSession(ctx context.Context, config *SessionConfig, logger model.Logger) (*engine.Session, error) {
	kvs, err := kvstore.NewFS(config.StateDir)
	if err != nil {
		return nil, err
	}
	proxyURL, err := parseProxyURL(config.ProxyURL)
	if err != nil {
		return nil, err
	}
	cfg := engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                kvs,
		Logger:                 logger,
		ProxyURL:               proxyURL, // nil if config.ProxyURL is ""
		SoftwareName:           config.SoftwareName,
		SoftwareVersion:        config.SoftwareVersion,
		TempDir:                config.TempDir,
		TorArgs:                config.TorArgs,
		TorBinary:              config.TorBinary,
		TunnelDir:              config.TunnelDir,
	}
	return engine.NewSession(ctx, cfg)
}
