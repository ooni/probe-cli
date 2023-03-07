package main

import (
	"context"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/legacy/assetsdir"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	softwareName    = "tinyooni"
	softwareVersion = version.Version
)

// newSession creates a new measurement session.
func newSession(ctx context.Context, globalOptions *GlobalOptions) (*engine.Session, error) {
	ooniDir := maybeGetOONIDir(globalOptions.HomeDir)
	if err := os.MkdirAll(ooniDir, 0700); err != nil {
		return nil, err
	}

	// We cleanup the assets files used by versions of ooniprobe
	// older than v3.9.0, where we started embedding the assets
	// into the binary and use that directly. This cleanup doesn't
	// remove the whole directory but only known files inside it
	// and then the directory itself, if empty. We explicitly discard
	// the return value as it does not matter to us here.
	_, _ = assetsdir.Cleanup(path.Join(ooniDir, "assets"))

	var (
		proxyURL *url.URL
		err      error
	)
	if globalOptions.Proxy != "" {
		proxyURL, err = url.Parse(globalOptions.Proxy)
		if err != nil {
			return nil, err
		}
	}

	kvstore2dir := filepath.Join(ooniDir, "kvstore2")
	kvstore, err := kvstore.NewFS(kvstore2dir)
	if err != nil {
		return nil, err
	}

	tunnelDir := filepath.Join(ooniDir, "tunnel")
	if err := os.MkdirAll(tunnelDir, 0700); err != nil {
		return nil, err
	}

	config := engine.SessionConfig{
		KVStore:             kvstore,
		Logger:              log.Log,
		ProxyURL:            proxyURL,
		SnowflakeRendezvous: globalOptions.SnowflakeRendezvous,
		SoftwareName:        softwareName,
		SoftwareVersion:     softwareVersion,
		TorArgs:             globalOptions.TorArgs,
		TorBinary:           globalOptions.TorBinary,
		TunnelDir:           tunnelDir,
	}
	if globalOptions.ProbeServicesURL != "" {
		config.AvailableProbeServices = []model.OOAPIService{{
			Address: globalOptions.ProbeServicesURL,
			Type:    "https",
		}}
	}

	sess, err := engine.NewSession(ctx, config)
	if err != nil {
		return nil, err
	}

	log.Debugf("miniooni temporary directory: %s", sess.TempDir())
	return sess, nil
}
