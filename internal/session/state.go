package session

//
// State of a bootstrapped session.
//

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/backendclient"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/sessionhttpclient"
	"github.com/ooni/probe-cli/v3/internal/sessionresolver"
	"github.com/ooni/probe-cli/v3/internal/tunnel"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// state is the bootstrapped [Session] state. Only the background
// goroutine is allowed to manipulate the [state].
type state struct {
	// backendClient is the backend client we're using.
	backendClient *backendclient.Client

	// checkIn is the most recently fetched check-in result.
	checkIn model.OptionalPtr[model.OOAPICheckInResult]

	// counter is the bytecounter we're using.
	counter *bytecounter.Counter

	// httpClient is the HTTP client we're using.
	httpClient model.HTTPClient

	// kvstore is the session's key-value store.
	kvstore model.KeyValueStore

	// location is the most recently resolved location.
	location model.OptionalPtr[geolocate.Results]

	// logger is the model.Logger we're using.
	logger model.Logger

	// resolver is the session resolver.
	resolver model.Resolver

	// softwareName is the software name to use.
	softwareName string

	// softwareVersion is the software version to use.
	softwareVersion string

	// tempDir is the session specific temporary directory.
	tempDir string

	// torBinary contains the location of the tor binary to use.
	torBinary string

	// torArgs should not be exposed here because we only
	// want to use it for bootstrapping tor.
	//torArgs []string

	// tunnelDir is the directory to use for tunnels.
	tunnelDir string

	// tunnel is the tunnel we're using.
	tunnel tunnel.Tunnel

	// userAgent is the user agent we're using when communicating
	// with the OONI backend (e.g., for the test helpers).
	userAgent string
}

// cleanup cleans the resources used by [state].
func (s *state) cleanup() {
	s.resolver.CloseIdleConnections()
	s.httpClient.CloseIdleConnections()
	s.tunnel.Stop()
	os.RemoveAll(s.tempDir)
}

// ErrEmptySoftwareName indicates the software name is empty.
var ErrEmptySoftwareName = errors.New("session: passed empty software name")

// ErrEmptySoftwareVersion indicates the software version is empty.
var ErrEmptySoftwareVersion = errors.New("session: passed empty software version")

// newState creates a new [state] instance.
func (s *Session) newState(ctx context.Context, req *BootstrapRequest) (*state, error) {
	if req.SoftwareName == "" {
		return nil, ErrEmptySoftwareName
	}
	if req.SoftwareVersion == "" {
		return nil, ErrEmptySoftwareVersion
	}

	logger := s.newLogger(req.VerboseLogging)

	// Implementation note: the context we receive from the caller limits the
	// whole lifetime of the tunnel we're going to create below. Because of
	// that, we're not allowed to cancel this context or add a timeout to it.

	ts := newTickerService(ctx, s)
	defer ts.stop()

	logger.Infof("bootstrap: creating key-value store at %s", req.StateDir)
	kvstore, err := kvstore.NewFS(req.StateDir)
	if err != nil {
		logger.Warnf("bootstrap: cannot create key-value store: %s", err.Error())
		return nil, err
	}

	logger.Infof("bootstrap: creating tunnels dir at %s", req.TunnelDir)
	if err := os.MkdirAll(req.TunnelDir, 0700); err != nil {
		logger.Warnf("bootstrap: cannot create tunnels dir: %s", err.Error())
		return nil, err
	}

	tempDir, err := stateNewTempDir(logger, req)
	if err != nil {
		// warning message already printed
		return nil, err
	}

	tunnel, err := newTunnel(ctx, logger, req)
	if err != nil {
		logger.Warnf("bootstrap: cannot create tunnel: %s", err.Error())
		return nil, err
	}

	state := newStateCannotFail(
		logger,
		kvstore,
		tunnel,
		tempDir,
		req,
	)
	return state, nil
}

// newStateCannotFail constructs a [state] once we have
// performed all operations that may fail.
func newStateCannotFail(
	logger model.Logger,
	kvstore model.KeyValueStore,
	tunnel tunnel.Tunnel,
	tempDir string,
	req *BootstrapRequest,
) *state {
	runtimex.Assert(logger != nil, "passed a nil logger")
	runtimex.Assert(kvstore != nil, "passed a nil kvstore")
	runtimex.Assert(tunnel != nil, "passed a nil tunnel")
	runtimex.Assert(tempDir != "", "passed an empty tempDir")
	runtimex.Assert(req != nil, "passed a nil req")

	logger.Infof("bootstrap: creating a session byte counter")
	counter := bytecounter.New()

	logger.Infof("bootstrap: creating a resolver for the session")
	resolver := &sessionresolver.Resolver{
		ByteCounter: counter,
		KVStore:     kvstore,
		Logger:      logger,
		ProxyURL:    tunnel.SOCKS5ProxyURL(), // possibly nil, which is OK
	}

	logger.Infof("bootstrap: creating an HTTP client for the session")
	httpClient := sessionhttpclient.New(&sessionhttpclient.Config{
		ByteCounter: counter,
		Logger:      logger,
		ProxyURL:    tunnel.SOCKS5ProxyURL(), // possibly nil, which is OK
		Resolver:    resolver,
	})

	logger.Infof("bootstrap: creating the default user-agent string")
	userAgent := fmt.Sprintf(
		"%s/%s ooniprobe-engine/%s",
		req.SoftwareName,
		req.SoftwareVersion,
		version.Version,
	)

	logger.Infof("bootstrap: creating an OONI backend client")
	backendClient := backendclient.New(&backendclient.Config{
		BaseURL:    nil, // use the default
		KVStore:    kvstore,
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  userAgent,
	})

	logger.Infof("bootstrap: complete")
	state := &state{
		backendClient:   backendClient,
		checkIn:         model.OptionalPtr[model.OOAPICheckInResult]{},
		counter:         counter,
		httpClient:      httpClient,
		kvstore:         kvstore,
		location:        model.OptionalPtr[geolocate.Results]{},
		logger:          logger,
		resolver:        resolver,
		softwareName:    req.SoftwareName,
		softwareVersion: req.SoftwareVersion,
		tempDir:         tempDir,
		torBinary:       req.TorBinary,
		tunnelDir:       req.TunnelDir,
		tunnel:          tunnel,
		userAgent:       userAgent,
	}
	return state
}

// stateNewTempDir creates a new temporary directory for [state].
func stateNewTempDir(logger model.Logger, req *BootstrapRequest) (string, error) {
	logger.Infof("bootstrap: creating temporary directory inside %s", req.TempDir)
	if err := os.MkdirAll(req.TempDir, 0700); err != nil {
		logger.Warnf("bootstrap: cannot create temporary directory root: %s", err.Error())
		return "", err
	}
	tempDir, err := os.MkdirTemp(req.TempDir, "")
	if err != nil {
		logger.Warnf("bootstrap: cannot create session temporary directory: %s", err.Error())
		return "", err
	}
	return tempDir, nil
}
