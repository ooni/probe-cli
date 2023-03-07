package main

import (
	"context"
	"errors"
	"net/url"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	errInvalidSessionId = errors.New("passed Session ID does not exist")
	mapSession          map[int64]*engine.Session
	idx                 int64 = 1
	mu                  sync.Mutex
)

func init() {
	taskRegistry["NewSession"] = &newSessionTaskRunner{}
	taskRegistry["DeleteSession"] = &deleteSessionTask{}
}

// newSessionOptions contains the request arguments for the NewSession task.
type newSessionOptions struct {
	ProxyUrl        string   `json:"ProxyUrl,omitempty"`
	StateDir        string   `json:"StateDir,omitempty"`
	SoftwareName    string   `json:"SoftwareName,omitempty"`
	SoftwareVersion string   `json:"SoftwareVersion,omitempty"`
	TempDir         string   `json:"TempDir,omitempty"`
	TorArgs         []string `json:"TorArgs,omitempty"`
	TorBinary       string   `json:"TorBinary,omitempty"`
	TunnelDir       string   `json:"TunnelDir,omitempty"`
}

// newSessionResponse is the response for the NewSession task.
type newSessionResponse struct {
	SessionId int64  `json:",omitempty"`
	Error     string `json:",omitempty"`
}

type newSessionTaskRunner struct{}

var _ taskRunner = &newSessionTaskRunner{}

// main implements taskRunner.main
func (tr *newSessionTaskRunner) main(ctx context.Context,
	emitter taskMaybeEmitter, req *request, resp *response) {
	logger := newTaskLogger(emitter)
	config := req.NewSession
	sess, err := newSession(ctx, &config, logger)
	if err != nil {
		resp.NewSession.Error = err.Error()
		logger.Warnf("engine: cannot create session: %s", err.Error())
		return
	}
	mu.Lock()
	defer mu.Unlock()
	resp.NewSession.SessionId = idx
	mapSession[idx] = sess
	idx++
}

// newSession creates a new *engine.Sessioncfg from the given config.
func newSession(ctx context.Context, config *newSessionOptions,
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

// parseProxyURL returns the proper proxy URL or nil if it's not configured.
func parseProxyURL(proxyURL string) (*url.URL, error) {
	if proxyURL == "" {
		return nil, nil
	}
	return url.Parse(proxyURL)
}

// deleteSessionOptions contains the request arguments for the DeleteSession task.
type deleteSessionOptions struct {
	SessionId int64 `json:",omitempty"`
}

// deleteSessionResponse is the response for the DeleteSession task.
type deleteSessionResponse struct {
	Error string `json:",omitempty"`
}

type deleteSessionTask struct{}

var _ taskRunner = &deleteSessionTask{}

// main implements taskRunner.main
func (tr *deleteSessionTask) main(ctx context.Context,
	emitter taskMaybeEmitter, req *request, resp *response) {
	sessionId := req.DeleteSession.SessionId
	// TODO(DecFox): add check to ensure we have a valid sessionId
	sess := mapSession[sessionId]
	if sess == nil {
		resp.DeleteSession.Error = errInvalidSessionId.Error()
		return
	}
	mu.Lock()
	defer mu.Unlock()
	mapSession[sessionId] = nil
}
