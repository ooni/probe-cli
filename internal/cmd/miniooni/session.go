package main

import (
	"context"
	"net/url"
	"os"
	"path/filepath"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

const (
	softwareName    = "miniooni"
	softwareVersion = version.Version
)

// newSessionOrPanic creates and starts a new session or panics on failure
func newSessionOrPanic(ctx context.Context, currentOptions Options,
	miniooniDir string, logger model.Logger) *engine.Session {
	var proxyURL *url.URL
	if currentOptions.Proxy != "" {
		proxyURL = mustParseURL(currentOptions.Proxy)
	}

	kvstore2dir := filepath.Join(miniooniDir, "kvstore2")
	kvstore, err := kvstore.NewFS(kvstore2dir)
	runtimex.PanicOnError(err, "cannot create kvstore2 directory")

	tunnelDir := filepath.Join(miniooniDir, "tunnel")
	err = os.MkdirAll(tunnelDir, 0700)
	runtimex.PanicOnError(err, "cannot create tunnelDir")

	config := engine.SessionConfig{
		KVStore:         kvstore,
		Logger:          logger,
		ProxyURL:        proxyURL,
		SoftwareName:    softwareName,
		SoftwareVersion: softwareVersion,
		TorArgs:         currentOptions.TorArgs,
		TorBinary:       currentOptions.TorBinary,
		TunnelDir:       tunnelDir,
	}
	if currentOptions.ProbeServicesURL != "" {
		config.AvailableProbeServices = []model.OOAPIService{{
			Address: currentOptions.ProbeServicesURL,
			Type:    "https",
		}}
	}

	sess, err := engine.NewSession(ctx, config)
	runtimex.PanicOnError(err, "cannot create measurement session")

	log.Debugf("miniooni temporary directory: %s", sess.TempDir())
	return sess
}

func lookupBackendsOrPanic(ctx context.Context, sess *engine.Session) {
	log.Info("Looking up OONI backends; please be patient...")
	err := sess.MaybeLookupBackendsContext(ctx)
	runtimex.PanicOnError(err, "cannot lookup OONI backends")
}

func lookupLocationOrPanic(ctx context.Context, sess *engine.Session) {
	log.Info("Looking up your location; please be patient...")
	err := sess.MaybeLookupLocationContext(ctx)
	runtimex.PanicOnError(err, "cannot lookup your location")

	log.Debugf("- IP: %s", sess.ProbeIP()) // make sure it does not appear in default logs
	log.Infof("- country: %s", sess.ProbeCC())
	log.Infof("- network: %s (%s)", sess.ProbeNetworkName(), sess.ProbeASNString())
	log.Infof("- resolver's IP: %s", sess.ResolverIP())
	log.Infof("- resolver's network: %s (%s)", sess.ResolverNetworkName(),
		sess.ResolverASNString())
}
