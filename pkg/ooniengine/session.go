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
	"github.com/ooni/probe-cli/v3/pkg/ooniengine/abi"
)

// parseProxyURL returns the proper proxy URL or nil if it's not configured.
func parseProxyURL(proxyURL string) (*url.URL, error) {
	if proxyURL == "" {
		return nil, nil
	}
	return url.Parse(proxyURL)
}

// newSession creates a new *engine.Session from the given Config.
func newSession(ctx context.Context, config *abi.SessionConfig, logger model.Logger) (*engine.Session, error) {
	// 🔥🔥🔥 Rule of thumb when reviewing protobuf code: if the code is using
	// the safe GetXXX accessors, it's good, otherwise it's not good
	kvs, err := kvstore.NewFS(config.GetStateDir())
	if err != nil {
		return nil, err
	}
	proxyURL, err := parseProxyURL(config.GetProxyUrl())
	if err != nil {
		return nil, err
	}
	cfg := engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                kvs,
		Logger:                 logger,
		ProxyURL:               proxyURL, // nil if config.ProxyURL is ""
		SoftwareName:           config.GetSoftwareName(),
		SoftwareVersion:        config.GetSoftwareVersion(),
		TempDir:                config.GetTempDir(),
		TorArgs:                config.GetTorArgs(),
		TorBinary:              config.GetTorBinary(),
		TunnelDir:              config.GetTunnelDir(),
	}
	return engine.NewSession(ctx, cfg)
}
