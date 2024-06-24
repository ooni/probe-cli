package wireguard

import (
	"context"
	"errors"
	"fmt"
	"net/netip"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"

	"github.com/amnezia-vpn/amneziawg-go/conn"
	"github.com/amnezia-vpn/amneziawg-go/device"
	"github.com/amnezia-vpn/amneziawg-go/tun"
	"github.com/amnezia-vpn/amneziawg-go/tun/netstack"
)

const (
	testName    = "wireguard"
	testVersion = "0.1.1"

	// defaultNameserver is the dns server using for resolving names inside the wg tunnel.
	defaultNameserver = "8.8.8.8"
)

var (
	ErrInputRequired    = targetloading.ErrInputRequired
	ErrInvalidInputType = targetloading.ErrInvalidInputType

	// TODO(ainghazal): fix after adding this error into targetloading
	ErrInvalidInput = errors.New("invalid input")
)

// Measurer performs the measurement.
type Measurer struct {
	events  *eventLogger
	options *wireguardOptions
	tnet    *netstack.Net
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{
		events:  newEventLogger(),
		options: &wireguardOptions{},
	}
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements model.ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	measurement := args.Measurement
	sess := args.Session
	zeroTime := measurement.MeasurementStartTimeSaved

	var err error

	// 0. fail if there is no richer input target.
	if args.Target == nil {
		return ErrInputRequired
	}

	// 1. setup tunnel after parsing options
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}

	// TODO(ainghazal): if the target is not public, substitute it with ASN?
	config, input := target.Options, target.URL
	if err := m.setupWireguardFromConfig(config); err != nil {
		// A failure at this point means that we are not able
		// to validate the minimal set of options that we need to probe an endpoint.
		// We abort the experiment and submit nothing.
		return err
	}

	// 2. create tunnel
	err = m.createTunnel(sess, zeroTime, config)

	testkeys := &TestKeys{
		Success: err == nil,
		Failure: measurexlite.NewFailure(err),
		URLGet:  make([]*URLGetResult, 0),
	}

	if config.PublicTarget {
		testkeys.Endpoint = m.options.endpoint
	} else {
		testkeys.Endpoint = input
	}

	testkeys.EndpointID = m.options.configurationHash()
	if config.PublicAmneziaParameters {
		// TODO(ainghazal): copy the parameters as testkeys
	}

	// 3. use tunnel
	if err == nil {
		sess.Logger().Info("Using the wireguard tunnel.")
		urlgetResult := m.urlget(defaultURLGetTarget, zeroTime, sess.Logger())
		testkeys.URLGet = append(testkeys.URLGet, urlgetResult)
		testkeys.NetworkEvents = m.events.log()
	}

	measurement.TestKeys = testkeys
	sess.Logger().Infof("%s", "Wireguard experiment done.")

	// NOTE: important to return nil to submit measurement.
	return nil
}

func (m *Measurer) setupWireguardFromConfig(config *Config) error {
	opts, err := newWireguardOptionsFromConfig(config)
	if err != nil {
		return err
	}
	if ok := opts.validate(); !ok {
		return fmt.Errorf("%w: %s", ErrInvalidInput, "cannot validate wireguard options")
	}
	m.options = opts
	return nil
}

func (m *Measurer) createTunnel(sess model.ExperimentSession, zeroTime time.Time, config *Config) error {
	sess.Logger().Info("wireguard: create tunnel")
	sess.Logger().Infof("endpoint: %s", m.options.endpoint)

	_, tnet, err := m.configureWireguardInterface(sess.Logger(), m.events, zeroTime, config)
	if err != nil {
		return err
	}
	m.tnet = tnet

	sess.Logger().Info("wireguard: create tunnel done")
	return nil
}

func (m *Measurer) configureWireguardInterface(
	logger model.Logger,
	eventlogger *eventLogger,
	zeroTime time.Time,
	config *Config) (tun.Device, *netstack.Net, error) {
	devTun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(m.options.ip)},
		[]netip.Addr{netip.MustParseAddr(m.options.ns)},
		1420)
	if err != nil {
		return nil, nil, err
	}

	dev := device.NewDevice(
		devTun,
		conn.NewDefaultBind(),
		newWireguardLogger(logger, eventlogger, config.Verbose, zeroTime, time.Since),
	)

	var ipcStr string

	opts := m.options

	ipcStr = `jc=` + opts.jc + `
jmin=` + opts.jmin + `
jmax=` + opts.jmax + `
s1=` + opts.s1 + `
s2=` + opts.s2 + `
h1=` + opts.h1 + `
h2=` + opts.h2 + `
h3=` + opts.h3 + `
h4=` + opts.h4 + `
private_key=` + opts.privKey + `
public_key=` + opts.pubKey + `
preshared_key=` + opts.presharedKey + `
endpoint=` + opts.endpoint + `
allowed_ip=0.0.0.0/0
`
	dev.IpcSet(ipcStr)

	err = dev.Up()
	if err != nil {
		return nil, nil, err
	}
	return devTun, tnet, nil
}

//
// logging utilities
//

// Event is a network event obtained by parsing wireguard logs.
type Event struct {
	EventType string  `json:"operation"`
	T         float64 `json:"t"`
}

func newEvent(etype string) *Event {
	return &Event{
		EventType: etype,
	}
}

type eventLogger struct {
	events []*Event
}

func newEventLogger() *eventLogger {
	return &eventLogger{events: make([]*Event, 0)}
}

func (el *eventLogger) append(e *Event) {
	el.events = append(el.events, e)
}

func (el *eventLogger) log() []*Event {
	return el.events
}

const (
	LOG_KEEPALIVE      = "Receiving keepalive packet"
	LOG_SEND_HANDSHAKE = "Sending handshake initiation"
	LOG_RECV_HANDSHAKE = "Received handshake response"

	EVT_RECV_KEEPALIVE      = "RECV_KEEPALIVE"
	EVT_SEND_HANDSHAKE_INIT = "SEND_HANDSHAKE_INIT"
	EVT_RECV_HANDSHAKE_RESP = "RECV_HANDSHAKE_RESP"
)

// newWireguardLogger looks at the strings logged by the wireguard
// implementation. It performs simple regex matching and then
// it appends the matchign Event in the passed eventLogger.
// This approach has some potential for brittleness (in the unlikely case
// that upstream wireguard codebase changes the emitted log lines),
// but adding typed log events to the wg codebase might prove to be a
// particularly time-consuming rewrite.
func newWireguardLogger(
	logger model.Logger,
	eventlogger *eventLogger,
	verbose bool,
	zeroTime time.Time,
	sinceFn func(time.Time) time.Duration) *device.Logger {
	verbosef := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)

		if verbose {
			logger.Debugf(msg)
		}

		// TODO(ainghazal): we might be interested in parsing additional events.
		if strings.Contains(msg, LOG_KEEPALIVE) {
			evt := newEvent(EVT_RECV_KEEPALIVE)
			evt.T = sinceFn(zeroTime).Seconds()
			eventlogger.append(evt)
			return
		}
		if strings.Contains(msg, LOG_SEND_HANDSHAKE) {
			evt := newEvent(EVT_SEND_HANDSHAKE_INIT)
			evt.T = sinceFn(zeroTime).Seconds()
			eventlogger.append(evt)
			return
		}
		if strings.Contains(msg, LOG_RECV_HANDSHAKE) {
			evt := newEvent(EVT_RECV_HANDSHAKE_RESP)
			evt.T = sinceFn(zeroTime).Seconds()
			eventlogger.append(evt)
			return
		}
	}
	errorf := func(format string, args ...any) {
		logger.Warnf(format, args...)
	}
	return &device.Logger{
		Verbosef: verbosef,
		Errorf:   errorf,
	}
}
