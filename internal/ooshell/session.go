package ooshell

//
// session.go
//
// Contains code to create a measurement session.
//

import (
	"context"
	"net/url"
	"os"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

// sessionDB wraps engine.Session and includes DB information.
type sessionDB struct {
	// id is the session ID
	id int64

	// Session is the underlying session
	*engine.Session
}

// ID returns the session's ID
func (sess *sessionDB) ID() int64 {
	return sess.id
}

// newSessionDB creates a new sessionDB instance.
func (env *Environ) newSessionDB(ctx context.Context) (*sessionDB, error) {
	sessID, err := env.DB.NewSession()
	if err != nil {
		return nil, err
	}
	sess, err := env.newEngineSession(ctx)
	env.DB.SetSessionBootstrapResult(sessID, err)
	if err != nil {
		return nil, err
	}
	return &sessionDB{id: sessID, Session: sess}, nil
}

// newEngineSession creates a new *engine.Session.
func (env *Environ) newEngineSession(ctx context.Context) (*engine.Session, error) {
	if err := os.MkdirAll(env.KVStoreDir, 0700); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(env.TunnelDir, 0700); err != nil {
		return nil, err
	}
	outch := make(chan *engine.Session)
	errch := make(chan error)
	go env.newSessionAsync(ctx, outch, errch)
	select {
	case sess := <-outch:
		return sess, nil
	case err := <-errch:
		return nil, err
	}
}

// newSessionAsync creates a new session asynchronously.
func (env *Environ) newSessionAsync(
	ctx context.Context, outch chan<- *engine.Session, errch chan<- error) {
	sess, err := env.newSessionWithoutLookups(ctx)
	if err != nil {
		errch <- err
		return
	}
	if err := sess.MaybeLookupLocationContext(ctx); err != nil {
		errch <- err
		return
	}
	if err := sess.MaybeLookupBackendsContext(ctx); err != nil {
		errch <- err
		return
	}
	outch <- sess
}

// newSessionWithoutLookups creates a new session but does perform lookups.
func (env *Environ) newSessionWithoutLookups(ctx context.Context) (*engine.Session, error) {
	proxyURL, err := env.proxyURL()
	if err != nil {
		return nil, err
	}
	kvstore, err := env.kvstore()
	if err != nil {
		return nil, err
	}
	config := engine.SessionConfig{
		KVStore:         kvstore,
		Logger:          env.Logger,
		ProxyURL:        proxyURL,
		SoftwareName:    env.SoftwareName,
		SoftwareVersion: env.SoftwareVersion,
		TorArgs:         env.TorArgs,
		TorBinary:       env.TorBinary,
		TunnelDir:       env.TunnelDir,
	}
	if env.ProbeServicesURL != "" {
		config.AvailableProbeServices = []model.Service{{
			Address: env.ProbeServicesURL,
			Type:    "https",
		}}
	}
	return engine.NewSession(ctx, config)
}

// proxyURL derives the proxy URL from the environment.
//
// Returns:
//
// - nil, nil if there is no configured proxy;
//
// - <*url.URL>, nil on success;
//
// - nil, <error> in case of failure.
func (env *Environ) proxyURL() (*url.URL, error) {
	if proxyURL := env.ProxyURL; proxyURL != "" {
		return url.Parse(env.ProxyURL)
	}
	return nil, nil
}

// kvstore returns the file-system-based KVStore
// instance that the OONI session should use.
func (env *Environ) kvstore() (engine.KVStore, error) {
	return kvstore.NewFS(env.KVStoreDir)
}
