// Package openvpn contains the openvpn experiment. This experiment
// measures the bootstrapping of an OpenVPN connection against a given remote.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-032-openvpn.md
package openvpn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"text/template"
	"time"

	"github.com/ooni/minivpn/extras/ping"
	"github.com/ooni/minivpn/vpn"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	// testName is the name of this experiment
	testName = "openvpn"

	// testVersion is the openvpn experiment version.
	testVersion = "0.0.1"

	// pingCount tells how many icmp echo requests to send.
	// pingCount = 10
	// TODO settle on standard.
	pingCount = 5

	// pingTarget is the target IP we used for pings.
	pingTarget = "8.8.8.8"

	// urlGrabURL is the URI we fetch to check web connectivity and egress IP.
	urlGrabURI = "https://api.ipify.org/?format=json"
)

// Config contains the experiment config.
type Config struct {
	SafeKey  string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCa   string `ooni:"ca to connect to the OpenVPN endpoint"`
	Cipher   string `ooni:"cipher to use"`
	Auth     string `ooni:"auth to use"`
	Compress string `ooni:"compression to use"`
}

// PingStats holds the results for a pinger run.
// TODO move the aggregates to summaryKeys?
type PingStats struct {
	MinRtt      float64   `json:"min_rtt"`
	MaxRtt      float64   `json:"max_rtt"`
	AvgRtt      float64   `json:"avg_rtt"`
	StdRtt      float64   `json:"std_rtt"`
	Rtts        []float64 `json:"rtts"`
	TTLs        []int     `json:"ttls"`
	PacketsRecv int       `json:"pkt_rcv"`
	PacketsSent int       `json:"pkt_snt"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// VPNLogs contains the bootstrap logs.

	// TODO pass the logger to the client??
	VPNLogs []string `json:"vpn_logs"`

	// MiniVPNVersion contains the version of the minivpn library used.
	MiniVPNVersion string `json:"minivpn_version"`

	// PingStats holds values for the aggregated stats of a ping.
	PingStats *PingStats `json:"ping_stats"`

	// Proto is the protocol used in the experiment.
	Proto string `json:"proto"`

	// Remote is the remote used in the experiment.
	Remote string `json:"remote"`

	// ...

	// PingTarget is the target we used for ping
	PingTarget string `json:"ping_target"`

	// just to capture something for now..
	Response string `json:"ip_query"`

	// Success is true when we reached the end of the test without errors.
	Success bool `json:"success"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config Config

	// vpnOptions is a minivpn.vpn.Options object with the parsed OpenVPN config options.
	vpnOptions *vpn.Options

	// rawDialer is the raw OpenVPN dialer
	rawDialer *vpn.RawDialer

	// conn is the vpn Client  net.Conn
	conn net.Conn
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

func parseStats(pinger *ping.Pinger) *PingStats {
	st := pinger.Statistics()
	pingStats := &PingStats{
		MinRtt:      st.MinRtt.Seconds(),
		MaxRtt:      st.MaxRtt.Seconds(),
		AvgRtt:      st.AvgRtt.Seconds(),
		StdRtt:      st.StdDevRtt.Seconds(),
		Rtts:        st.Rtts,
		TTLs:        st.TTLs,
		PacketsRecv: st.PacketsRecv,
		PacketsSent: st.PacketsSent,
	}
	return pingStats
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
	experiment, err := vpnExperimentFromURI(string(measurement.Input))

	dialer, err := m.setup(ctx, experiment, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of time.
		return err
	}
	m.rawDialer = dialer

	m.registerExtensions(measurement)

	const maxRuntime = 600 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()

	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second) // this is copied from some other experiment to allow a progress display; reuse.
	defer ticker.Stop()

	go m.bootstrap(ctx, sess, tkch)

	select {
	case tk := <-tkch:
		measurement.TestKeys = tk
		break
	}
	tk := measurement.TestKeys.(*TestKeys)

	//
	// All ready now. Now we can begin the experiment itself.
	//
	// 1. ping
	//

	pinger := ping.NewFromSharedConnection(pingTarget, m.conn)
	pinger.Count = pingCount
	tk.PingTarget = pingTarget

	err = pinger.Run(ctx)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		return err
	}

	tk.PingStats = parseStats(pinger)

	//
	// 2. urlgrab
	//

	d := vpn.NewTunDialer(m.rawDialer)

	client := http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
		},
	}
	resp, err := client.Get(urlGrabURI)
	if err != nil {
		sess.Logger().Warnf("openvpn: failed urlgrab: %s", err)
		tk.Failure = tracex.NewFailure(fmt.Errorf("%w: %s", ErrURLGrab, err))
		return err
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sess.Logger().Warnf("openvpn: failed urlgrab: %s", err)
		tk.Failure = tracex.NewFailure(fmt.Errorf("%w: %s", ErrURLGrab, err))
		return err
	}
	tk.Response = string(body)
	sess.Logger().Info("openvpn: all tests ok")
	tk.Success = true
	return nil
}

// setup prepares for running the openvpn experiment. Returns a minivpn dialer on success.
// Returns an error on failure.
func (m *Measurer) setup(ctx context.Context, exp *VPNExperiment, logger model.Logger) (*vpn.RawDialer, error) {
	exp.Config = &VPNExperimentConfig{}
	exp.Config.Auth = m.config.Auth
	exp.Config.Cipher = m.config.Cipher
	exp.Config.Compress = m.config.Compress

	// TODO(ainghazal): capture cert validation errors into test failures ---
	ca, _ := extractBase64Blob(m.config.SafeCa)
	cert, _ := extractBase64Blob(m.config.SafeCert)
	key, _ := extractBase64Blob(m.config.SafeKey)

	exp.Config.Ca = ca
	exp.Config.Cert = cert
	exp.Config.Key = key

	tmp, err := os.CreateTemp("", "vpn-")
	if err != nil {
		return nil, err
	}

	t := template.New("openvpnConfig")
	t, err = t.Parse(vpnConfigTemplate)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, exp)

	if err != nil {
		return nil, err
	}

	tmp.Write(buf.Bytes())
	o, err := vpn.ParseConfigFile(tmp.Name())
	if err != nil {
		return nil, err
	}

	logger.Infof("Using Config File: %s", tmp.Name())
	// TODO(ainghazal): defer delete of the file after DEBUG

	m.vpnOptions = o
	// TODO(ainghazal): pass context to dialer
	raw := vpn.NewRawDialer(o)
	return raw, nil
}

var ErrURLGrab = errors.New("urlgrab")

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, sess model.ExperimentSession,
	out chan<- *TestKeys) {
	tk := &TestKeys{
		BootstrapTime:  0,
		Failure:        nil,
		Proto:          protoToString(m.vpnOptions.Proto),
		Remote:         net.JoinHostPort(m.vpnOptions.Remote, m.vpnOptions.Port),
		MiniVPNVersion: getMiniVPNVersion(),
	}
	sess.Logger().Info("openvpn: bootstrapping openvpn connection")
	defer func() {
		out <- tk
	}()

	s := time.Now()
	conn, err := m.rawDialer.DialContext(ctx)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
	}
	m.conn = conn
	m.rawDialer.ReuseClient(conn)
	tk.BootstrapTime = time.Now().Sub(s).Seconds()

	sess.Logger().Info("openvpn: bootstrapping done")
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
	errInvalidTestKeysType = errors.New("openvpn: invalid test keys type")

	//errNilTestKeys indicates that the test keys are nil.
	errNilTestKeys = errors.New("openvpn: nil test keys")
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

func getMiniVPNVersion() string {
	bi, ok := debug.ReadBuildInfo()
	if !ok {
		return "unknown"
	}
	for _, dep := range bi.Deps {
		p := strings.Split(dep.Path, "/")
		if p[len(p)-1] == "minivpn" {
			return dep.Version
		}
	}
	return ""
}
