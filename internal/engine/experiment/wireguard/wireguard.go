// Package wireguard contains the wireguard experiment. This experiment
// measures the bootstrapping of an WireGuard connection against a given remote.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-040-wireguard.md

package wireguard

import (
	"bufio"
	"context"
	"errors"
	"log"
	"net/netip"
	"os"
	"strings"
	"sync"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (

	// testName is the name of this experiment
	testName = "wireguard"

	// testVersion is the wireguard experiment version.
	testVersion = "0.0.2"

	// pingCount tells how many icmp echo requests to send.
	pingCount = 10

	// successLossThreshold will mark ICMP pings as successful if loss is below this number.
	successLossThreshold = 0.5

	// pingTarget is the target IP we used for pings.
	pingTarget = "8.8.8.8"

	// pingTargetNZ is a target IP for a mirror with known geolocation.
	pingTargetNZ = "163.7.134.112"

	pingExtraWaitSeconds = 2

	// defaultNameserver is the dns server using for resolving names inside the wg tunnel.
	defaultNameserver = "8.8.8.8"

	// urlGrabURL is the URI we fetch to check web connectivity and egress IP.
	urlGrabURI = "https://api.ipify.org/?format=json"

	// googleURI is self-explanatory.
	googleURI = "https://www.google.com/"
)

var (
	icmpTimeoutSeconds = 10
	errBadBootstrap    = "bad_bootstrap"
	errPingError       = "ping_error"
	errBadReplyType    = "ping_bad_reply_type"
	errBadReply        = "ping_bad_reply"
)

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config  Config
	options *options
	device  device.Device
	tun     tun.Device
	tnet    *netstack.Net
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

// registerExtensions registers the extensions used by this experiment.
func (m *Measurer) registerExtensions(measurement *model.Measurement) {
	// currently none
}

type netstackDialer struct {
	*netstack.Net
}

// CloseIdleConnections satisfies model.Dialer.ClosCloseIdleConnections
func (nd netstackDialer) CloseIdleConnections() {
	// Probably want to call some netstack.Net method in here...
}

// Run runs the experiment with the specified context, session,
// measurement, and experiment calbacks. This method should only
// return an error in case the experiment could not run (e.g.,
// a required input is missing). Otherwise, the code should just
// set the relevant OONI error inside of the measurement and
// return nil. This is important because the caller may not submit
// the measurement if this method returns an error.
func (m *Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	// FIXME get config from input + safeXXX entries.
	config := string(measurement.Input)
	err := m.setup(ctx, config, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of time.
		return err
	}
	m.registerExtensions(measurement)

	const maxRuntime = 600 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	go m.bootstrap(ctx, sess, tkch)
	select {
	case tk := <-tkch:
		measurement.TestKeys = tk
		break
	}
	tk := measurement.TestKeys.(*TestKeys)
	wg := new(sync.WaitGroup)
	tk.Pings = []*PingResult{}

	sendBlockingPing(wg, m.tun, m.tnet, pingTarget, tk)
	//TODO(ainghazal): get gateway ip
	//sendBlockingPing(wg, m.tunnel, remoteVPNGateway, tk)
	sendBlockingPing(wg, m.tun, m.tnet, pingTargetNZ, tk)

	wantedICMP := 1
	goodICMP := 0
	for _, p := range tk.Pings[:wantedICMP] {
		if p.PacketsSent == 0 {
			break
		}
		loss := 1 - float32(p.PacketsRecv)/float32(p.PacketsSent)
		if loss < successLossThreshold {
			goodICMP += 1
		}
	}
	if goodICMP == wantedICMP {
		tk.SuccessICMP = true
	}

	sess.Logger().Infof("wireguard: urlgrab stage")

	targetURLs := []string{urlGrabURI}

	if len(m.config.URLs) != 0 {
		urls := strings.Split(m.config.URLs, ",")
		targetURLs = append(targetURLs, urls...)
	}

	urlgetterConfig := urlgetter.Config{
		Dialer: netstackDialer{m.tnet},
	}

	for _, uri := range targetURLs {
		wg.Add(1)
		go func() {
			defer wg.Done()

			g := urlgetter.Getter{
				Config:  urlgetterConfig,
				Session: sess,
				Target:  uri,
			}
			urlgetTk, _ := g.Get(context.Background())
			tk.Requests = append(tk.Requests, urlgetTk.Requests...)
		}()
		wg.Wait()
	}

	goodURLGrabs := 0
	for _, r := range tk.Requests {
		if r.Failure == nil {
			goodURLGrabs += 1
		}
	}
	if goodURLGrabs != 0 {
		tk.SuccessURLGrab = true
	}

	sess.Logger().Info("openvpn: all tests ok")
	tk.Success = true
	return nil
}

// setup prepares for running the wireguard experiment. Returns an error on failure.
func (m *Measurer) setup(ctx context.Context, config string, logger model.Logger) error {
	// TODO: use model.VPNConfig for common attrs
	// ie, parse the input vpn://proto.provider/ ...
	o, err := getOptionsFromConfig(m.config)
	if err != nil {
		return err
	}
	m.options = o
	return nil
}

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, sess model.ExperimentSession,
	out chan<- *TestKeys) {
	tk := &TestKeys{
		BootstrapTime: 0,
		Failure:       nil,
		Provider:      "unknown",
		Proto:         testName,
		Transport:     "udp",
		Remote:        m.options.endpoint,
		Obfuscation:   "none",
	}
	sess.Logger().Info("wireguard: bootstrapping experiment")
	defer func() {
		out <- tk
	}()

	s := time.Now()
	tun, tnet, err := doWireguardBootstrap(m.options)
	if err != nil {
		tk.Failure = &errBadBootstrap
		return
	}

	// this bootstrap is only local interface setup, so maybe it does not make sense to
	// include it in measurements.
	tk.BootstrapTime = time.Now().Sub(s).Seconds()
	m.tun = tun
	m.tnet = tnet
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

var (
	// errInvalidTestKeysType indicates the test keys type is invalid.
	errInvalidTestKeysType = errors.New("wireguard: invalid test keys type")

	//errNilTestKeys indicates that the test keys are nil.
	errNilTestKeys = errors.New("wireguard: nil test keys")
)

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	testkeys, good := measurement.TestKeys.(*TestKeys)
	if !good {
		return nil, errInvalidTestKeysType
	}
	if testkeys == nil {
		return nil, errNilTestKeys
	}
	return SummaryKeys{IsAnomaly: testkeys.Failure != nil}, nil
}

func doWireguardBootstrap(o *options) (tun.Device, *netstack.Net, error) {
	devTun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(o.ip)},
		[]netip.Addr{netip.MustParseAddr(o.ns)},
		1420)
	if err != nil {
		log.Panic(err)
	}
	dev := device.NewDevice(devTun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
	dev.IpcSet(`private_key=` + o.privKey + `
public_key=` + o.pubKey + `
preshared_key=` + o.presharedKey + `
endpoint=` + o.endpoint + `
allowed_ip=0.0.0.0/0
`)

	err = dev.Up()
	if err != nil {
		return nil, nil, err
	}
	return devTun, tnet, nil
}

func getLinesFromFile(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	lines := make([]string, 0)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	err = scanner.Err()
	if err != nil {
		return nil, err
	}
	return lines, nil
}
