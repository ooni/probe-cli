package tortunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/cretz/bine/tor"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/shellx"
)

// Start attempts to start a tor [Tunnel].
func Start(ctx context.Context, config *Config) (*Tunnel, error) {
	// obtain the logger to use
	logger := config.logger()

	// determine the tunnel directory to use
	tunnelDir, cleanupTunnelDir, err := config.tunnelDir(logger)
	if err != nil {
		return nil, err
	}
	stateDir := filepath.Join(tunnelDir, "tor")
	logger.Infof("tortunnel: stateDir: %s", stateDir)

	// determine the log file to use
	logFile := filepath.Join(stateDir, "tor.log")
	logger.Infof("tortunnel: logFile: %s", logFile)

	// cleanup any leftovers from a previous invocation, if needed
	maybeCleanupStateDir(logger, stateDir, logFile)

	// setup command line arguments.
	extraArgs := append([]string{}, config.TorArgs...)
	extraArgs = append(extraArgs, "Log")
	extraArgs = append(extraArgs, fmt.Sprintf(`notice file %s`, logFile))
	logCommandLineArguments(logger, extraArgs)

	// TODO(bassosimone): handle snowflake

	// generate the start configuration
	torStartConf, err := newTorStartConf(config, stateDir, extraArgs)
	if err != nil {
		cleanupTunnelDir()
		return nil, err
	}

	// obtain the dependencies we should use.
	deps := config.dependencies()

	// start the tor process
	instance, err := deps.Start(ctx, torStartConf)
	if err != nil {
		cleanupTunnelDir()
		return nil, err
	}

	// make sure we close the running process on Close
	instance.StopProcessOnClose = true

	// obtain and emit the tor version
	protoInfo, err := deps.TorControlProtocolInfo(instance)
	if err != nil {
		instance.Close()
		cleanupTunnelDir()
		return nil, err
	}
	logger.Infof("tortunnel: tor version: %s", protoInfo.TorVersion)
	select {
	case config.TorVersion <- protoInfo.TorVersion:
	default:
	}

	// wait for the bootstrap to complete
	startBootstrap := time.Now()
	if err := deps.TorEnableNetwork(ctx, instance, true); err != nil {
		instance.Close()
		cleanupTunnelDir()
		return nil, err
	}
	bootstrapTime := time.Since(startBootstrap)
	logger.Infof("tortunnel: bootstrap time: %v", bootstrapTime)

	// get the proxy URL
	proxyURL, err := getProxyURL(deps, instance)
	if err != nil {
		instance.Close()
		cleanupTunnelDir()
		return nil, err
	}

	// TODO(bassosimone): we still need to set the name correctly

	// construct a tunnel instance
	tunnel := &Tunnel{
		bootstrapTime:        bootstrapTime,
		instance:             instance,
		maybeDeleteTunnelDir: cleanupTunnelDir,
		name:                 "",
		proxy:                proxyURL,
		stopOnce:             sync.Once{},
	}

	return tunnel, nil
}

// bootstrap performs the bootstrap and returns its duration.
func bootstrap(ctx context.Context, config *Config, instance *tor.Tor) (time.Duration, error) {
}

// ErrCannotGetSOCKS5ProxyURL indicates we cannot get the SOCKS5 proxy URL.
var ErrCannotGetSOCKS5ProxyURL = errors.New("tortunnel: cannot get SOCKS5 proxy URL")

// getProxyURL attempts to obtain the proxy URL.
func getProxyURL(deps *Dependencies, instance *tor.Tor) (*url.URL, error) {
	// Adapted from <https://git.io/Jfc7N>
	info, err := deps.TorControlGetInfo(instance, "net/listeners/socks")
	if err != nil {
		return nil, err
	}
	if len(info) != 1 || info[0].Key != "net/listeners/socks" {
		instance.Close()
		return nil, ErrCannotGetSOCKS5ProxyURL
	}
	proxyAddress := info[0].Val
	if strings.HasPrefix(proxyAddress, "unix:") {
		instance.Close()
		return nil, ErrCannotGetSOCKS5ProxyURL
	}
	proxyURL := &url.URL{Scheme: "socks5", Host: proxyAddress}
	return proxyURL, nil
}

// maybeCleanupStateDir removes stale files inside the stateDir.
func maybeCleanupStateDir(logger model.Logger, stateDir, logFile string) {
	maybeRemoveFile(logger, logFile)
	maybeRemoveWithGlob(logger, filepath.Join(stateDir, "torrc-*"))
	maybeRemoveWithGlob(logger, filepath.Join(stateDir, "control-port-*"))
}

// maybeRemoveWithGlob globs and removes files.
func maybeRemoveWithGlob(logger model.Logger, pattern string) {
	files, _ := filepath.Glob(pattern)
	for _, file := range files {
		maybeRemoveFile(logger, file)
	}
}

// maybeRemoveFile removes a file if needed.
func maybeRemoveFile(logger model.Logger, file string) {
	logger.Infof("tortunnel: rm -f %s", file)
	os.Remove(file)
}

// logCommandLineArguments logs the command line arguments we're using.
func logCommandLineArguments(logger model.Logger, args []string) {
	quoted := shellx.QuotedCommandLineUnsafe("tor", args...)
	logger.Infof("tortunnel: command line: %s", quoted)
}
