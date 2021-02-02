// Package psiphonx is a wrapper around the psiphon-tunnel-core.
package psiphonx

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

// Session is the way in which this package sees a Session.
type Session interface {
	NewOrchestraClient(ctx context.Context) (model.ExperimentOrchestraClient, error)
	TempDir() string
}

// Dependencies contains dependencies for Start
type Dependencies interface {
	MkdirAll(path string, perm os.FileMode) error
	RemoveAll(path string) error
	Start(ctx context.Context, config []byte,
		workdir string) (*clientlib.PsiphonTunnel, error)
}

type defaultDependencies struct{}

func (defaultDependencies) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (defaultDependencies) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (defaultDependencies) Start(
	ctx context.Context, config []byte, workdir string) (*clientlib.PsiphonTunnel, error) {
	return clientlib.StartTunnel(ctx, config, "", clientlib.Parameters{
		DataRootDirectory: &workdir}, nil, nil)
}

// Config contains the settings for Start. The empty config object implies
// that we will be using default settings for starting the tunnel.
type Config struct {
	// Dependencies contains dependencies for Start.
	Dependencies Dependencies

	// WorkDir is the directory where Psiphon should store
	// its configuration database.
	WorkDir string
}

// Tunnel is a psiphon tunnel
type Tunnel struct {
	tunnel   *clientlib.PsiphonTunnel
	duration time.Duration
}

func makeworkingdir(config Config) (string, error) {
	const testdirname = "oonipsiphon"
	workdir := filepath.Join(config.WorkDir, testdirname)
	if err := config.Dependencies.RemoveAll(workdir); err != nil {
		return "", err
	}
	if err := config.Dependencies.MkdirAll(workdir, 0700); err != nil {
		return "", err
	}
	return workdir, nil
}

// Start starts the psiphon tunnel.
func Start(
	ctx context.Context, sess Session, config Config) (*Tunnel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err() // simplifies unit testing this code
	default:
	}
	if config.Dependencies == nil {
		config.Dependencies = defaultDependencies{}
	}
	if config.WorkDir == "" {
		config.WorkDir = sess.TempDir()
	}
	clnt, err := sess.NewOrchestraClient(ctx)
	if err != nil {
		return nil, err
	}
	configJSON, err := clnt.FetchPsiphonConfig(ctx)
	if err != nil {
		return nil, err
	}
	workdir, err := makeworkingdir(config)
	if err != nil {
		return nil, err
	}
	start := time.Now()
	tunnel, err := config.Dependencies.Start(ctx, configJSON, workdir)
	if err != nil {
		return nil, err
	}
	stop := time.Now()
	return &Tunnel{tunnel: tunnel, duration: stop.Sub(start)}, nil
}

// Stop is an idempotent method that shuts down the tunnel
func (t *Tunnel) Stop() {
	if t != nil {
		t.tunnel.Stop()
	}
}

// SOCKS5ProxyURL returns the SOCKS5 proxy URL.
func (t *Tunnel) SOCKS5ProxyURL() (proxyURL *url.URL) {
	if t != nil {
		proxyURL = &url.URL{
			Scheme: "socks5",
			Host: net.JoinHostPort(
				"127.0.0.1", fmt.Sprintf("%d", t.tunnel.SOCKSProxyPort)),
		}
	}
	return
}

// BootstrapTime returns the bootstrap time
func (t *Tunnel) BootstrapTime() (duration time.Duration) {
	if t != nil {
		duration = t.duration
	}
	return
}
