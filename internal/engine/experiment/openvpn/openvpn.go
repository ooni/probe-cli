// Package openvpn contains the openvpn experiment. This experiment
// measures the bootstrapping of an OpenVPN connection against a given remote.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-032-openvpn.md
package openvpn

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"strings"
	"sync"
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
	testVersion = "0.0.6"

	// pingCount tells how many icmp echo requests to send.
	pingCount = 10

	// pingExtraWaitSeconds tells how many grace seconds to wait after
	// last ping in train.
	pingExtraWaitSeconds = 2

	// pingTarget is the target IP we used for pings (high-availability, replicated clusters).
	pingTarget = "8.8.8.8"

	// pingTargetFaraway is a target IP for a mirror with known geolocation.
	pingTargetFaraway = "163.7.134.112"

	// urlGrabURL is the URI we fetch to check web connectivity and egress IP.
	urlGrabURI = "https://api.ipify.org/?format=json"
)

var (
	bootstrapError = "bootstrap-error"
	urlgrabError   = "urlgrab-error"
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

// PingReply is a single response in the ping sequence.
type PingReply struct {
	Seq int     `json:"seq"`
	TTL int     `json:"ttl"`
	Rtt float64 `json:"rtt"`
}

// PingResult holds the results for a pinger run.
type PingResult struct {
	Target      string      `json:"target"`
	Sequence    []PingReply `json:"sequence"`
	PacketsRecv int         `json:"pkt_rcv"`
	PacketsSent int         `json:"pkt_snt"`
	MinRtt      int64       `json:"min_rtt"`
	MaxRtt      int64       `json:"max_rtt"`
	AvgRtt      int64       `json:"avg_rtt"`
	StdRtt      int64       `json:"std_rtt"`
	// FIXME unsure about this, but I think I want to know if
	// the measurements timed out for each ping, or it's another kind of
	// error (ie, instrumental, uncovered cases etc).
	Error *string `json:"error"`
}

// URLURLGrabResult holds the results for a urlgrab run.
// TODO we should store more things here:
// fetch time
// response code
// (this serves another purpose: check for geofencing etc...)
type URLGrabResult struct {
	URI        string  `json:"uri"`
	Response   *string `json:"response"`
	FetchTime  float64 `json:"fetch_time_ms"`
	StatusCode int     `json:"status"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	//
	// Keys that will serve as primary keys.
	//

	// Provider is the entity that controls the endpoints.
	Provider string `json:"provider"`

	// Proto is the protocol used in the experiment (openvpn in this case).
	Proto string `json:"vpn_protocol"`

	// Transport is the transport protocol (tcp, udp).
	Transport string `json:"transport"`

	// Remote is the remote used in the experiment (ip:addr).
	Remote string `json:"remote"`

	// Obfuscation is the kind of obfuscation used, if any.
	Obfuscation string `json:"obfuscation"`

	//
	// Other keys
	//

	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Stage captures a uint16 event measuring the progress of the VPN connection.
	Stage uint16 `json:"stage"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// Error is one of `null`, `"bootstrap-error"`, `"timeout-reached"`,
	// and `"unknown-error"`.
	// TODO(ainghazal): make sure all are covered.
	Error *string `json:"error"`

	// Pings holds an array for aggregated stats of each ping.
	Pings []*PingResult `json:"pings"`

	// URLGrab holds an array for urlgrab results.
	URLGrab []*URLGrabResult `json:"urlgrab"`

	// MiniVPNVersion contains the version of the minivpn library used.
	MiniVPNVersion string `json:"minivpn_version"`

	// Success is true when we reached the end of the test without errors.
	Success bool `json:"success"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config Config

	// vpnOptions is a minivpn.vpn.Options object with the parsed OpenVPN config options.
	vpnOptions *vpn.Options

	// tunnel is the vpn.Client
	tunnel *vpn.Client
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

// pingTimeout returns the timeout set on each pinger train.
func pingTimeout() time.Duration {
	return time.Second * (pingCount + pingExtraWaitSeconds)
}

func doSinglePing(wg *sync.WaitGroup, conn net.Conn, target string, tk *TestKeys) {
	defer wg.Done()

	pinger := ping.NewFromSharedConnection(target, conn)
	pinger.Count = pingCount
	pinger.Timeout = pingTimeout()

	conn.SetReadDeadline(time.Now().Add(time.Second * 60))
	err := pinger.Run(context.Background())

	log.Println("ping done!")

	tk.Pings = append(tk.Pings, parseStats(pinger, target))
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
	}
}

func sendBlockingPing(wg *sync.WaitGroup, conn net.Conn, target string, tk *TestKeys) {
	wg.Add(1)
	go doSinglePing(wg, conn, target, tk)
	wg.Wait()
	log.Printf("ping train sent to %s ----", target)
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
	if err != nil {
		return err
	}
	tunnel, err := m.setup(experiment, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of time.
		return err
	}
	m.tunnel = tunnel
	m.registerExtensions(measurement)

	const maxRuntime = 600 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()

	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second) // this is copied from some other experiment to allow a progress display; reuse.
	defer ticker.Stop()

	go m.bootstrap(ctx, sess, experiment, tkch)

	select {
	case tk := <-tkch:
		measurement.TestKeys = tk
		break
	}
	tk := measurement.TestKeys.(*TestKeys)

	/*
	 defer func() {
	 	if m.tunnel != nil {
	 		m.tunnel.Close()
	 	}
	 }()
	*/

	if tk.Failure != nil {
		// bootstrap error
		return nil
	}

	remoteVPNGateway := m.tunnel.RemoteAddr().String()

	//
	// All ready now. Now we can begin the experiment itself.
	//
	// 1. ping external target, gateway and a third location.
	//

	wg := new(sync.WaitGroup)
	tk.Pings = []*PingResult{}
	sendBlockingPing(wg, m.tunnel, pingTarget, tk)
	sendBlockingPing(wg, m.tunnel, remoteVPNGateway, tk)
	sendBlockingPing(wg, m.tunnel, pingTargetFaraway, tk)

	//
	// 2. urlgrab
	//

	// TODO(ainghazal): it's cleaner to pass a closure to DialContext in
	// the client transport. I just need to reset the read timeoutb before
	// reusing the conn.
	log.Println("stage: urlgrab")

	m.tunnel.SetReadDeadline(time.Now().Add(time.Second * 60))
	d := vpn.NewTunDialer(m.tunnel)

	client := http.Client{
		Transport: &http.Transport{
			DialContext: d.DialContext,
		},
	}

	urlgrabResult := &URLGrabResult{
		URI:      urlGrabURI,
		Response: nil,
	}

	fetchStart := time.Now()
	resp, err := client.Get(urlGrabURI)
	if err != nil {
		sess.Logger().Warnf("openvpn: failed urlgrab: %s", err)
		tk.Failure = tracex.NewFailure(err)
		tk.Error = &urlgrabError
		tk.URLGrab = append(tk.URLGrab, urlgrabResult)
		return nil
	}
	urlgrabResult.StatusCode = resp.StatusCode
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		sess.Logger().Warnf("openvpn: failed urlgrab: %s", err)
		tk.Failure = tracex.NewFailure(err)
		tk.Error = &urlgrabError
		tk.URLGrab = append(tk.URLGrab, urlgrabResult)
		return nil
	}

	rb := string(body)
	urlgrabResult.Response = &rb
	urlgrabResult.FetchTime = float64(time.Now().Sub(fetchStart).Microseconds() / 1000.0)

	tk.URLGrab = append(tk.URLGrab, urlgrabResult)
	sess.Logger().Info("openvpn: all tests ok")
	tk.Success = true
	return nil
}

// setup prepares for running the openvpn experiment. Returns a minivpn dialer on success.
// Returns an error on failure.
func (m *Measurer) setup(exp *model.VPNExperiment, logger model.Logger) (*vpn.Client, error) {
	exp.Config = &model.VPNConfig{}
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
	opt, err := vpn.NewOptionsFromFilePath(tmp.Name())
	if err != nil {
		return nil, err
	}

	logger.Infof("Using Config File: %s", tmp.Name())
	// TODO(ainghazal): defer delete of the file after DEBUG

	m.vpnOptions = opt
	tunnel := vpn.NewClientFromOptions(opt)
	return tunnel, nil
}

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, sess model.ExperimentSession,
	experiment *model.VPNExperiment,
	out chan<- *TestKeys) {
	tk := &TestKeys{
		Provider:       experiment.Provider,
		Proto:          testName,
		Transport:      protoToString(m.vpnOptions.Proto),
		Remote:         net.JoinHostPort(m.vpnOptions.Remote, m.vpnOptions.Port),
		Obfuscation:    experiment.Obfuscation,
		URLGrab:        []*URLGrabResult{},
		MiniVPNVersion: getMiniVPNVersion(),
		BootstrapTime:  0,
		Failure:        nil,
		Error:          nil,
	}
	sess.Logger().Info("openvpn: bootstrapping openvpn connection")
	defer func() {
		out <- tk
	}()

	s := time.Now()

	// ---------------------------------------------------
	// TODO use step-by-step to get a trace for the dialer
	// trace := measurexlite.NewTrace(index, zeroTime)
	// ---------------------------------------------------

	vpnEventChan := make(chan uint16, 100)
	m.tunnel.EventListener = vpnEventChan

	go func() {
		for {
			select {
			case stage := <-vpnEventChan:
				tk.Stage = stage
			}
		}
	}()

	err := m.tunnel.Start(ctx)
	tk.BootstrapTime = time.Now().Sub(s).Seconds()
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		tk.Error = &bootstrapError
		sess.Logger().Info("openvpn: bootstrapping failed")
		return
	}
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

// parseStats accepts a pointer to a Pinger struct and a target string, and returns
// an pointer to a PingResult with all the fields filled.
func parseStats(pinger *ping.Pinger, target string) *PingResult {
	st := pinger.Statistics()
	replies := []PingReply{}
	for _, r := range st.Replies {
		replies = append(replies, PingReply{
			Seq: r.Seq,
			Rtt: r.Rtt,
			TTL: r.TTL,
		})
	}
	pingStats := &PingResult{
		Target:      target,
		PacketsRecv: st.PacketsRecv,
		PacketsSent: st.PacketsSent,
		Sequence:    replies,
		MinRtt:      st.MinRtt.Milliseconds(),
		MaxRtt:      st.MaxRtt.Milliseconds(),
		AvgRtt:      st.AvgRtt.Milliseconds(),
		StdRtt:      st.StdDevRtt.Milliseconds(),
	}
	return pingStats
}
