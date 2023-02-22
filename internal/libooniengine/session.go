package main

import (
	"context"
	"encoding/json"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type sessConfig struct {
	ProxyUrl        string   `json:"ProxyUrl,omitempty"`
	StateDir        string   `json:"StateDir,omitempty"`
	SoftwareName    string   `json:"SoftwareName,omitempty"`
	SoftwareVersion string   `json:"SoftwareVersion,omitempty"`
	TempDir         string   `json:"TempDir,omitempty"`
	TorArgs         []string `json:"TorArgs,omitempty"`
	TorBinary       string   `json:"TorBinary,omitempty"`
	TunnelDir       string   `json:"TunnelDir,omitempty"`
}

func init() {
	taskRegistry["NewSession"] = &newSessionTaskRunner{}
}

type newSessionTaskRunner struct{}

var _ taskRunner = &newSessionTaskRunner{}

func (tr *newSessionTaskRunner) main(ctx context.Context,
	emitter taskMaybeEmitter, args []byte) {
	logger := newTaskLogger(emitter)
	var config *sessConfig
	if err := json.Unmarshal(args, config); err != nil {
		logger.Warnf("engine: cannot deserialize arguments: %s", err.Error())
		return
	}
	// TODO(DecFox): we are ignoring the session here but we want to use this for further tasks.
	_, err := newSession(ctx, config, logger)
	if err != nil {
		logger.Warnf("engine: cannot create session: %s", err.Error())
		return
	}
}

// newSession creates a new *engine.Sessioncfg from the given config.
func newSession(ctx context.Context, config *sessConfig,
	logger model.Logger) (*engine.Session, error) {
	kvs, err := kvstore.NewFS(config.StateDir)
	if err != nil {
		return nil, err
	}
	// Note: while we are passing a proxyUrl here, we do not bootstrap any tunnels in
	// this function.
	proxyURL, err := parseProxyURL(config.ProxyUrl)
	if err != nil {
		return nil, err
	}
	cfg := &engine.SessionConfig{
		AvailableProbeServices: []model.OOAPIService{},
		KVStore:                kvs,
		Logger:                 logger,
		ProxyURL:               proxyURL, // nil if cfg.ProxyURL is ""
		SoftwareName:           config.SoftwareName,
		SoftwareVersion:        config.SoftwareVersion,
		TempDir:                config.TempDir,
		TorArgs:                config.TorArgs,
		TorBinary:              config.TorBinary,
		TunnelDir:              config.TunnelDir,
	}
	return engine.NewSessionWithoutTunnel(ctx, cfg)
}

// parseProxyURL returns the proper proxy URL or nil if it's not cfgured.
func parseProxyURL(proxyURL string) (*url.URL, error) {
	if proxyURL == "" {
		return nil, nil
	}
	return url.Parse(proxyURL)
}
