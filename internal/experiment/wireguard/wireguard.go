package wireguard

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"

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
	ErrInputRequired = errors.New("input is required")
)

// Measurer performs the measurement.
type Measurer struct {
	config    Config
	rawconfig []byte
	options   options

	events   *eventLogger
	testName string
	tnet     *netstack.Net
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return m.testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements model.ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	measurement := args.Measurement
	sess := args.Session
	zeroTime := measurement.MeasurementStartTimeSaved

	var err error

	// 0. obtain the richer input target, config, and input or panic
	if args.Target == nil {
		return ErrInputRequired
	}

	// 1. setup (parse config file)
	target := args.Target.(*Target)

	// TODO(ainghazal): process the input when the backend hands us one.
	config, _ := target.Options, target.URL

	if err := m.setupConfig(config); err != nil {
		return err
	}

	// 2. create tunnel
	err = m.createTunnel(sess, zeroTime)

	testkeys := &TestKeys{
		Success: err == nil,
		Failure: measurexlite.NewFailure(err),
		URLGet:  make([]*URLGetResult, 0),
	}

	// 3. use tunnel
	if err == nil {
		sess.Logger().Info("Using the wireguard tunnel.")
		urlgetResult := m.urlget(zeroTime, sess.Logger())
		testkeys.URLGet = append(testkeys.URLGet, urlgetResult)
		testkeys.NetworkEvents = m.events.log()
	}

	measurement.TestKeys = testkeys
	sess.Logger().Infof("%s", "Wireguard experiment done.")

	// NOTE: important to return nil to submit measurement.
	return nil
}

func (m *Measurer) setupConfig(config *Config) error {
	opts, err := getOptionsFromConfig(config)
	if err != nil {
		return err
	}
	m.options = opts
	return nil
}

func (m *Measurer) createTunnel(sess model.ExperimentSession, zeroTime time.Time) error {
	sess.Logger().Info("wireguard: create tunnel")
	sess.Logger().Infof("endpoint: %s", m.options.endpoint)

	_, tnet, err := m.configureWireguardInterface(sess.Logger(), m.events, zeroTime)
	if err != nil {
		return err
	}
	m.tnet = tnet

	sess.Logger().Info("wireguard: create tunnel done")
	return nil
}

func newURLResultFromError(url string, zeroTime time.Time, start float64, err error) *URLGetResult {
	return &URLGetResult{
		URL:     url,
		T0:      start,
		T:       time.Since(zeroTime).Seconds(),
		Failure: measurexlite.NewFailure(err),
		Error:   err.Error(),
	}
}

func newURLResultWithStatusCode(url string, zeroTime time.Time, start float64, statusCode int, body []byte) *URLGetResult {
	return &URLGetResult{
		ByteCount:  len(body),
		URL:        url,
		T0:         start,
		T:          time.Since(zeroTime).Seconds(),
		StatusCode: statusCode,
	}
}

func (m *Measurer) urlget(zeroTime time.Time, logger model.Logger) *URLGetResult {
	url := "https://info.cern.ch/"
	client := http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			DialContext:         m.tnet.DialContext,
			TLSHandshakeTimeout: 30 * time.Second,
		}}

	start := time.Since(zeroTime).Seconds()
	r, err := client.Get(url)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Warnf("urlget error: %v", err.Error())
		return newURLResultFromError(url, zeroTime, start, err)
	}
	defer r.Body.Close()

	return newURLResultWithStatusCode(url, zeroTime, start, r.StatusCode, body)
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return Measurer{
		events:   newEventLogger(),
		options:  options{},
		testName: testName,
	}
}

func (m *Measurer) configureWireguardInterface(
	logger model.Logger,
	eventlogger *eventLogger,
	zeroTime time.Time) (tun.Device, *netstack.Net, error) {
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
		newWireguardLogger(logger, eventlogger, m.config.Verbose, zeroTime),
	)

	var ipcStr string

	// If the rawconfig string has content, it means that we
	// did not bother to pass every option separatedly, so we assume
	// we got a valid config file. This might be dangerous, so think twice
	// about enforcing proper validation of the configuration file.
	if len(m.rawconfig) > 0 {
		ipcStr = string(m.rawconfig)
	} else {
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
	}
	dev.IpcSet(ipcStr)

	err = dev.Up()
	if err != nil {
		return nil, nil, err
	}
	return devTun, tnet, nil
}

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

func newWireguardLogger(
	logger model.Logger,
	eventlogger *eventLogger,
	verbose bool,
	zeroTime time.Time) *device.Logger {
	verbosef := func(format string, args ...any) {
		msg := fmt.Sprintf(format, args...)

		if verbose {
			logger.Debugf(msg)
		}

		// TODO(ainghazal): we might be interested in parsing other type of events here.
		if strings.Contains(msg, "Receiving keepalive packet") {
			evt := newEvent("RECV_KEEPALIVE")
			evt.T = time.Since(zeroTime).Seconds()
			eventlogger.append(evt)
			return
		}
		if strings.Contains(msg, "Sending handshake initiation") {
			evt := newEvent("SEND_HANDSHAKE_INIT")
			evt.T = time.Since(zeroTime).Seconds()
			eventlogger.append(evt)
			return
		}
		if strings.Contains(msg, "Received handshake response") {
			evt := newEvent("RECV_HANDSHAKE_RESP")
			evt.T = time.Since(zeroTime).Seconds()
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
