package ootunnel

import (
	"context"
	_ "embed" // for embedding
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"time"

	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

//go:embed psiphon.json
var configFile []byte

// psiphonCloser converts the psiphon Tunnel to io.Closer
type psiphonCloser struct {
	t *clientlib.PsiphonTunnel
}

// Close implements io.Closer.
func (c *psiphonCloser) Close() error {
	c.t.Stop()
	return nil
}

func (b *Broker) getMkdirAll() func(path string, perms fs.FileMode) error {
	if b.mkdirAll != nil {
		return b.mkdirAll
	}
	return os.MkdirAll
}

// newPsiphon starts a psiphon tunnel.
func (b *Broker) newPsiphon(ctx context.Context, config *Config) (Tunnel, error) {
	stateDir := config.StateDir
	start := time.Now()
	if err := b.getMkdirAll()(stateDir, 0700); err != nil {
		return nil, err
	}
	pt, err := clientlib.StartTunnel(ctx, configFile, "", clientlib.Parameters{
		DataRootDirectory: &stateDir,
	}, nil, nil)
	stop := time.Now()
	if err != nil {
		return nil, err
	}
	return &tunnelish{
		bootstrapTime:         stop.Sub(start),
		deleteStateDirOnClose: config.DeleteStateDirOnClose,
		name:                  Psiphon,
		proxyURL: &url.URL{
			Scheme: "socks5",
			Host:   fmt.Sprintf("127.0.0.1:%d", pt.SOCKSProxyPort),
		},
		stateDir: config.StateDir,
		t:        &psiphonCloser{pt},
	}, nil
}
