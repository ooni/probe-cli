package miniengine

//
// Measurement session
//

import (
	"errors"
	"fmt"
	"os"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/probeservices"
)

// SessionConfig contains configuration for a [Session]. The zero value is
// invalid; please, initialize all the fields marked as MANDATORY.
type SessionConfig struct {
	// SoftwareName is the MANDATORY name of the application
	// that will be using the new [Session].
	SoftwareName string `json:"software_name"`

	// SoftwareVersion is the MANDATORY version of the application
	// that will be using the new [Session].
	SoftwareVersion string `json:"software_version"`

	// StateDir is the MANDATORY directory where to store state
	// information required by a [Session].
	StateDir string `json:"state_dir"`

	// TempDir is the MANDATORY directory inside which the [Session] shall
	// store temporary files deleted when we close the [Session].
	TempDir string `json:"temp_dir"`

	// TunnelDir is the MANDATORY directory where the [Session] shall store
	// persistent data regarding circumvention tunnels.
	TunnelDir string `json:"tunnel_dir"`

	// Verbose OPTIONALLY configures the [Session] logger to be verbose.
	Verbose bool `json:"verbose"`
}

// ErrSessionConfig indicates that the [SessionConfig] is invalid.
var ErrSessionConfig = errors.New("invalid SessionConfig")

// check checks whether the [SessionConfig] is valid.
func (sc *SessionConfig) check() error {
	if sc.SoftwareName == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "SoftwareName is empty")
	}
	if sc.SoftwareVersion == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "SoftwareVersion is empty")
	}
	if sc.StateDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "StateDir is empty")
	}
	if sc.TempDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "TempDir is empty")
	}
	if sc.TunnelDir == "" {
		return fmt.Errorf("%w: %s", ErrSessionConfig, "TunnelDir is empty")
	}
	return nil
}

// mkdirAll ensures all the required directories exist.
func (sc *SessionConfig) mkdirAll() error {
	if err := os.MkdirAll(sc.StateDir, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(sc.TempDir, 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(sc.TunnelDir, 0700); err != nil {
		return err
	}
	return nil
}

// Session is a measurement session. The zero value is invalid; please
// create a new instance using the [NewSession] factory.
type Session struct {
	// closeJustOnce ensures we close this [Session] just once.
	closeJustOnce sync.Once

	// emitter is the [emitter] to use.
	emitter chan *Event

	// logger is the [model.Logger] to use.
	logger model.Logger

	// mu provides mutual exclusion.
	mu sync.Mutex

	// softwareName is the software name.
	softwareName string

	// softwareVersion is the software version.
	softwareVersion string

	// stateDir is the directory containing state.
	stateDir string

	// state contains the optional state.
	state optional.Value[*engineSessionState]

	// tempDir is the temporary directory root.
	tempDir string

	// tunnelDir is the directory containing tunnel state.
	tunnelDir string
}

// engineSessionState contains the state associated with an [engine.Session].
type engineSessionState struct {
	// checkIn contains the check-in API response.
	checkIn *model.OOAPICheckInResult

	// geoloc contains the geolocation.
	geoloc *Location

	// psc is the [probeservices.Client] to use.
	psc *probeservices.Client

	// sess is the underlying [engine.Session].
	sess *engine.Session
}

// NewSession creates a new [Session] instance.
func NewSession(config *SessionConfig) (*Session, error) {
	// check whether the [SessionConfig] is valid.
	if err := config.check(); err != nil {
		return nil, err
	}

	// make sure all the required directories exist.
	if err := config.mkdirAll(); err != nil {
		return nil, err
	}

	// create the base event emitter
	const buffer = 1024
	emitter := make(chan *Event, buffer)

	// create a logger using the base event emitter
	logger := newLoggerEmitter(emitter, config.Verbose)

	// assemble and return a session.
	sess := &Session{
		closeJustOnce:   sync.Once{},
		emitter:         emitter,
		logger:          logger,
		mu:              sync.Mutex{},
		softwareName:    config.SoftwareName,
		softwareVersion: config.SoftwareVersion,
		stateDir:        config.StateDir,
		state:           optional.None[*engineSessionState](),
		tempDir:         config.TempDir,
		tunnelDir:       config.TunnelDir,
	}
	return sess, nil
}

// ErrNoBootstrap indicates that you did not bootstrap the [Session].
var ErrNoBootstrap = errors.New("bootstrap the session first")

// CheckInResult returns the check-in API result.
func (s *Session) CheckInResult() (*model.OOAPICheckInResult, error) {
	// make sure this method is synchronized
	defer s.mu.Unlock()
	s.mu.Lock()

	// handle the case where there's no state.
	if s.state.IsNone() {
		return nil, ErrNoBootstrap
	}

	// return the underlying value
	return s.state.Unwrap().checkIn, nil
}

// GeolocateResult returns the geolocation result.
func (s *Session) GeolocateResult() (*Location, error) {
	// make sure this method is synchronized
	defer s.mu.Unlock()
	s.mu.Lock()

	// handle the case where there's no state.
	if s.state.IsNone() {
		return nil, ErrNoBootstrap
	}

	// return the underlying value
	return s.state.Unwrap().geoloc, nil
}

// Close closes the [Session]. This function is safe to call multiple
// times. We'll close underlying resources on the first invocation and
// otherwise do nothing for subsequent invocations.
func (s *Session) Close() (err error) {
	s.closeJustOnce.Do(func() {
		// make sure the cleanup is synchronized.
		defer s.mu.Unlock()
		s.mu.Lock()

		// handle the case where there is no state.
		if s.state.IsNone() {
			return
		}

		// obtain the underlying state
		state := s.state.Unwrap()

		// replace with empty state
		s.state = optional.None[*engineSessionState]()

		// close the underlying session
		err = state.sess.Close()
	})
	return err
}
