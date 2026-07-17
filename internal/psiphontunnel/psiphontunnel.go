package psiphontunnel

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/ooni/probe-cli/v3/internal/feature/psiphonfeat"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// TunnelInfo contains information about the tunnel.
type TunnelInfo struct {
	// BootstrapID is the unique ID of this bootstrap.
	BootstrapID int64

	// BootstrapTime is the time of the last bootstrap regardless of
	// whether it completed successfully.
	BootstrapTime time.Duration

	// LastBootstrapError is the error occurred during the latest bootstrap.
	LastBootstrapError error

	// Up is true if the tunnel is up.
	Up bool

	// currentProxyURL is the URL to use the tunnel as a proxy.
	currentProxyURL string
}

// ErrTunnelDown indicates that the tunnel is down.
var ErrTunnelDown = errors.New("psiphontunnel: tunnel ios down")

// ProxyURL returns the tunnel proxy URL.
func (ti *TunnelInfo) ProxyURL() (*url.URL, error) {
	if !ti.Up {
		return nil, ErrTunnelDown
	}
	return url.Parse(ti.currentProxyURL)
}

// Starter provides the starting-the-psiphon-tunnel functionality.
type Starter interface {
	Start(ctx context.Context, config []byte, workdir string) (psiphonfeat.Tunnel, error)
}

// PsiphonStarter is the [Starter] that starts a psiphon tunnel using psiphon libraries as
// opposed to other providers which may just be mocks.
type PsiphonStarter struct{}

var _ Starter = &PsiphonStarter{}

// Start implements Starter.
func (*PsiphonStarter) Start(ctx context.Context, config []byte, workdir string) (psiphonfeat.Tunnel, error) {
	return psiphonfeat.Start(ctx, config, workdir)
}

// Service is the service that manages the psiphon tunnel.
type Service struct {
	// configch is the channel for sending the psiphon config to the service.
	configch chan *configMessage

	// querych is the channel for sending a query request to the service.
	querych chan *queryMessage

	// startch is the channel for sending a request to start the tunnel to the service.
	startch chan *startMessage

	// stopch is the channel for sending a request to stop the tunnel to the service.
	stopch chan *stopMessage
}

// Singleton is the singleton managing psiphon tunnels.
var Singleton = StartService()

// StartService starts the psiphon tunnel [*Service] in the background.
func StartService() *Service {
	svc := &Service{
		configch: make(chan *configMessage),
		querych:  make(chan *queryMessage),
		startch:  make(chan *startMessage),
		stopch:   make(chan *stopMessage),
	}
	go svc.loop()
	return svc
}

// configMessage provides configuration to the psiphon service.
type configMessage struct {
	// config contains the psiphon config.
	config []byte

	// dir contains the directory where to write tunnel data.
	dir string

	// errch is used to return potential errors.
	errch chan error
}

// queryMessage is a request to obtain information about the tunnel.
type queryMessage struct {
	infoch chan *TunnelInfo
}

// startMessage tells the psiphon service it should start or restart the tunnel.
type startMessage struct {
	// abortch must be closed by StartTunnel when returning early to
	// potentially interrupt an ongoing starting attempt.
	abortch chan any

	// errch is used to return potential errors.
	errch chan error

	// starter is the starter to use.
	starter Starter
}

// stopMessage tells the psiphon service it should stop a tunnel.
type stopMessage struct {
	// errch is used to return potential errors.
	errch chan error
}

// SendConfig sends the psiphon config to the service.
func (svc *Service) SendConfig(ctx context.Context, config []byte, baseDir string) error {
	// prepare the message for the service
	m := &configMessage{
		config: config,
		dir:    filepath.Join(baseDir, "psiphon"),
		errch:  make(chan error, 1),
	}

	// attempt to send the message to the service.
	select {
	case svc.configch <- m:
		// ok

	case <-ctx.Done():
		return ctx.Err()
	}

	// attempt to receive the result of sending the config.
	select {
	case err := <-m.errch:
		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}

// Query queries the service about the current existing tunnel.
func (svc *Service) Query(ctx context.Context) (*TunnelInfo, error) {
	// prepare the message for the service
	m := &queryMessage{
		infoch: make(chan *TunnelInfo),
	}

	// attempt to send the message to the service.
	select {
	case svc.querych <- m:
		// ok

	case <-ctx.Done():
		return nil, ctx.Err()
	}

	// attempt to receive the result of sending the config.
	select {
	case info := <-m.infoch:
		return info, nil

	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// StartTunnel asks the service to start or restart the psiphon tunnel.
func (svc *Service) StartTunnel(ctx context.Context, starter Starter) error {
	// prepare the message for the service
	m := &startMessage{
		abortch: make(chan any),
		errch:   make(chan error, 1),
		starter: starter,
	}

	// make sure we close abortch if we bail early so to stop establishing the tunnel
	defer close(m.abortch)

	// attempt to send the message to the service.
	select {
	case svc.startch <- m:
		// ok

	case <-ctx.Done():
		return ctx.Err()
	}

	// attempt to receive the result of starting the tunnel.
	select {
	case err := <-m.errch:
		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}

// StopTunnel asks the service to stop the tunnel of it is running.
func (svc *Service) StopTunnel(ctx context.Context) error {
	// prepare the message for the service
	m := &stopMessage{
		errch: make(chan error, 1),
	}

	// attempt to send the message to the service.
	select {
	case svc.stopch <- m:
		// ok

	case <-ctx.Done():
		return ctx.Err()
	}

	// attempt to receive the result of stopping the tunnel.
	select {
	case err := <-m.errch:
		return err

	case <-ctx.Done():
		return ctx.Err()
	}
}

// loop runs the main loop of the psiphon service.
func (svc *Service) loop() {
	state := &psiphonServiceLoopState{
		bootstrapID:          0,
		config:               optional.Value[[]byte]{},
		currentProxyURL:      optional.Value[string]{},
		dir:                  optional.Value[string]{},
		lastBootstrapFailure: optional.Value[error]{},
		lastBootstrapTime:    optional.Value[time.Duration]{},
		tunnel:               optional.Value[psiphonfeat.Tunnel]{},
	}

	for {
		select {
		case msg := <-svc.configch:
			state.onConfig(msg)

		case msg := <-svc.querych:
			state.onQuery(msg)

		case msg := <-svc.stopch:
			state.onStop(msg)

		case msg := <-svc.startch:
			state.onStart(msg)
		}
	}
}

// psiphonServiceLoopState contains stated used by the psiphon service loop.
type psiphonServiceLoopState struct {
	// bootstrapID is the unique ID of this bootstrap.
	bootstrapID int64

	// config contains the psiphon tunnel config.
	config optional.Value[[]byte]

	// currentProxyURL is the current proxy URL or empty.
	currentProxyURL optional.Value[string]

	// dir is the directory where we should operate.
	dir optional.Value[string]

	// lastBootstrapFailure is the failure that occurred during
	// the last bootstrap attempt or empty.
	lastBootstrapFailure optional.Value[error]

	// lastBootstrapTime is the time it took to the last bootstrap
	// attempt to complete, regardless of whether it succeded.
	lastBootstrapTime optional.Value[time.Duration]

	// tunnel is the psiphon tunnel proper.
	tunnel optional.Value[psiphonfeat.Tunnel]
}

// onConfig is called when we receive a config message.
func (psl *psiphonServiceLoopState) onConfig(msg *configMessage) {
	// save values
	psl.dir = optional.Some(msg.dir)
	psl.config = optional.Some(msg.config)

	// send response
	// channel is buffered so we're not blocking
	msg.errch <- nil
}

// onQuery is called when we receive a query message.
func (psl *psiphonServiceLoopState) onQuery(msg *queryMessage) {
	// prepare the tunnel info struct
	info := &TunnelInfo{
		BootstrapID:        psl.bootstrapID,
		BootstrapTime:      psl.lastBootstrapTime.UnwrapOr(0),
		currentProxyURL:    psl.currentProxyURL.UnwrapOr(""),
		LastBootstrapError: psl.lastBootstrapFailure.UnwrapOr(nil),
		Up:                 !psl.tunnel.IsNone(),
	}

	// send response
	// channel is buffered so we're not blocking
	msg.infoch <- info
}

// onStop is called when we receive a stop message.
func (psl *psiphonServiceLoopState) onStop(msg *stopMessage) {
	// stop if possible
	psl.maybeStop()

	// send response
	// channel is buffered so we're not blocking
	msg.errch <- nil
}

// maybeStop stops the tunnel unless it has already been stopped.
func (psl *psiphonServiceLoopState) maybeStop() {
	// invalidate the current proxy URL
	psl.currentProxyURL = optional.None[string]()

	// stop if possible
	if t := psl.tunnel.UnwrapOr(nil); t != nil {
		t.Stop()
		psl.tunnel = optional.None[psiphonfeat.Tunnel]()
	}
}

// ErrTunnelStart indicates an error with the psiphon service configuration.
var ErrTunnelStart = errors.New("psiphontunnel: cannot start tunnel")

// onStart is invoked when we receive a start message.
func (psl *psiphonServiceLoopState) onStart(msg *startMessage) {
	// stop if possible
	psl.maybeStop()

	// make sure we have a dir
	if psl.dir.IsNone() {
		// channel is buffered so we're not blocking
		msg.errch <- fmt.Errorf("%w: you did not configure a tunnel dir", ErrTunnelStart)
		return
	}

	// make sure we have a config
	if psl.config.IsNone() {
		// channel is buffered so we're not blocking
		msg.errch <- fmt.Errorf("%w: you did not provide a psiphon config", ErrTunnelStart)
		return
	}

	// attempt to create state dir
	if err := os.MkdirAll(psl.dir.Unwrap(), 0755); err != nil {
		// channel is buffered so we're not blocking
		msg.errch <- fmt.Errorf("%w: %s", ErrTunnelStart, err.Error())
		return
	}

	// create a context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make sure that, if msg.aborted is closed, we abort starting the tunnel
	go func() {
		select {
		case <-ctx.Done():
		case <-msg.abortch:
			cancel()
		}
	}()

	// increment the bootstrap ID
	psl.bootstrapID++

	// start the actual tunnel
	started := time.Now()
	tun, err := msg.starter.Start(ctx, psl.config.Unwrap(), psl.dir.Unwrap())

	// record the last bootstrap time
	psl.lastBootstrapTime = optional.Some(time.Since(started))

	// check for errors
	if err != nil {
		// record the failure
		psl.lastBootstrapFailure = optional.Some(err)

		// channel is buffered so we're not blocking
		msg.errch <- fmt.Errorf("%w: %s", ErrTunnelStart, err.Error())
		return
	}

	// record the success
	psl.lastBootstrapFailure = optional.None[error]()

	// remember that we have a running tunnel
	psl.tunnel = optional.Some(tun)

	// remember the proxy URL
	URL := &url.URL{
		Scheme: "socks5",
		Host:   net.JoinHostPort("127.0.0.1", strconv.Itoa(tun.GetSOCKSProxyPort())),
		Path:   "/",
	}
	psl.currentProxyURL = optional.Some(URL.String())

	// tell the user we succeded
	// channel is buffered so we're not blocking
	msg.errch <- nil
}
