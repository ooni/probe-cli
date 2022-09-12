// Package openvpn contains the openvpn experiment. This experiment
// measures the bootstrapping of an OpenVPN connection against a given remote,
// a series of ICMP pings, and a series of url page fetches through the tunnel.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-032-openvpn.md
package openvpn

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/ooni/minivpn/extras/ping"
	"github.com/ooni/minivpn/vpn"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	// testName is the name of this experiment
	testName = "openvpn"

	// testVersion is the openvpn experiment version.
	testVersion = "0.0.11"

	// pingCount tells how many icmp echo requests to send.
	pingCount = 10

	// pingExtraWaitSeconds tells how many grace seconds to wait after
	// last ping in train.
	pingExtraWaitSeconds = 2

	// pingTarget is the target IP we used for pings (high-availability, replicated clusters).
	pingTarget = "8.8.8.8"

	// pingTargetNZ is a target IP for a mirror with known geolocation.
	pingTargetNZ = "163.7.134.112"

	// urlGrabURL is the URI we fetch to check web connectivity and egress IP.
	urlGrabURI = "https://api.ipify.org/?format=json"

	// googleURI is self-explanatory.
	googleURI = "https://www.google.com/"
)

var (
	bootstrapError = "bootstrap-error"
	localCredsFile = "ooni-vpn-creds"
)

// Config contains the experiment config.
type Config struct {
	SafeKey        string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert       string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCa         string `ooni:"ca to connect to the OpenVPN endpoint"`
	Cipher         string `ooni:"cipher to use"`
	Auth           string `ooni:"auth to use"`
	Compress       string `ooni:"compression to use"`
	SafeLocalCreds bool   `ooni:"whether to use local credentials for the given provider"`
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
	MinRtt      float64     `json:"min_rtt"`
	MaxRtt      float64     `json:"max_rtt"`
	AvgRtt      float64     `json:"avg_rtt"`
	StdRtt      float64     `json:"std_rtt"`
	Error       *string     `json:"error"`
}

// Stage captures a uint16 event measuring the progress of the VPN connection.
type Stage struct {
	OpID      uint16  `json:"op_id"`
	Operation string  `json:"operation"`
	Time      float64 `json:"t"`
}

func newStage(st uint16, t time.Duration) Stage {
	var s string
	switch st {
	case vpn.EventReady:
		s = "ready"
	case vpn.EventDialDone:
		s = "dial_done"
	case vpn.EventReset:
		s = "reset"
	case vpn.EventTLSConn:
		s = "tls_conn"
	case vpn.EventTLSHandshake:
		s = "tls_handshake"
	case vpn.EventTLSHandshakeDone:
		s = "tls_handshake_done"
	case vpn.EventDataInitDone:
		s = "data_init"
	case vpn.EventHandshakeDone:
		s = "vpn_handshake_done"
	default:
		s = "unknown"
	}
	return Stage{
		OpID:      st,
		Operation: s,
		Time:      toMs(t),
	}
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

	// Stages is a sequence of stages with their corresponding timestamp.
	Stages []Stage `json:"stages"`

	// Dial connect traces a TCP connection for the vpn dialer (null for UDP transport).
	DialConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// Error is one of `null`, `"bootstrap-error"`, `"timeout-reached"`,
	// and `"unknown-error"`.
	// TODO(ainghazal): make sure all are covered.
	Error *string `json:"error"`

	// Pings holds an array for aggregated stats of each ping.
	Pings []*PingResult `json:"pings"`

	// Requests archive an arbitrary number of http requests done through the tunnel.
	Requests []*measurex.ArchivalHTTPRoundTripEvent `json:"requests"`

	// MiniVPNVersion contains the version of the minivpn library used.
	MiniVPNVersion string `json:"minivpn_version"`

	// TODO(ainghazal): implement
	// Obfs4Version contains the version of the obfs4 library used.
	Obfs4Version string `json:"obfs4_version"`

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

	// tmpConfigFile is the temporary file passwd to openvpn
	tmpConfigFile string
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

	err := pinger.Run(context.Background())
	pingResult := parseStats(pinger, target)
	if err != nil {
		e := err.Error()
		pingResult.Error = &e
	}
	tk.Pings = append(tk.Pings, pingResult)
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
	defer func() {
		err := os.Remove(filepath.Join(os.TempDir(), localCredsFile))
		if err != nil {
			sess.Logger().Infof(err.Error())
		}
		err = os.Remove(m.tmpConfigFile)
		if err != nil {
			sess.Logger().Infof(err.Error())
		}
	}()
	experiment, err := vpnExperimentFromURI(string(measurement.Input))
	if err != nil {
		return err
	}
	tunnel, err := m.setup(experiment, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of
		// time, we probably should log something more specific.
		return err
	}
	m.tunnel = tunnel
	m.registerExtensions(measurement)

	const maxRuntime = 600 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()

	tkch := make(chan *TestKeys)
	go m.bootstrap(ctx, sess, experiment, tkch)

	select {
	case tk := <-tkch:
		measurement.TestKeys = tk
		break
	}
	tk := measurement.TestKeys.(*TestKeys)

	/*
		// TODO(ainghazal): this is the right thing, but I think I need to add the ability
		// to cleanly shut down the device in the tunnel first - otherwise there's an extra error
		// on the logs (I think because the goroutines keep trying to copy data).
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

	//
	// 1. ping external target, gateway and a third location.
	//

	wg := new(sync.WaitGroup)
	tk.Pings = []*PingResult{}

	// TODO(ainghazal): for the sake of reducing experimental bias, we
	// should randomize the order of the following function calls. But that
	// is going to make parsing the data a bit harder, unless we convene on
	// a given idx.
	sendBlockingPing(wg, m.tunnel, pingTarget, tk)
	sendBlockingPing(wg, m.tunnel, remoteVPNGateway, tk)
	sendBlockingPing(wg, m.tunnel, pingTargetNZ, tk)

	//
	// 2. urlgrab
	//

	sess.Logger().Infof("openvpn: urlgrab stage")

	vpnDialer := vpn.NewTunDialer(m.tunnel)

	db := &measurex.MeasurementDB{}
	mx := measurex.NewMeasurerWithDefaultSettings()
	txp := measurex.WrapHTTPTransport(
		time.Now(),
		db,
		&txpTCP{&http.Transport{DialContext: vpnDialer.DialContext}},
		100) // this should be enough to get the html lang attribute
	clnt := &http.Client{
		Transport: txp,
		Jar:       measurex.NewCookieJar(),
	}

	targetURLs := []string{urlGrabURI, googleURI}

	for _, uri := range targetURLs {
		wg.Add(1)
		go func() {
			defer wg.Done()
			u, _ := url.Parse(uri)

			m.tunnel.SetReadDeadline(time.Now().Add(time.Second * 60))
			resp, _ := mx.HTTPClientGET(ctx, clnt, u)
			if resp != nil {
				resp.Body.Close()
			}
		}()
		wg.Wait()
	}

	tk.Requests = append(tk.Requests, measurex.NewArchivalHTTPRoundTripEventList(db.AsMeasurement().HTTPRoundTrip)...)

	sess.Logger().Info("openvpn: all tests ok")
	tk.Success = true
	return nil
}

// txpTCP implements model.HTTPTransport
type txpTCP struct {
	*http.Transport
}

func (t *txpTCP) Network() string {
	return "tcp"
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
	exp.Config.LocalCreds = m.config.SafeLocalCreds

	if exp.Config.LocalCreds {
		// TODO create temp file and pass it as localCreds (string)
		tmpCreds, err := os.Create(filepath.Join(os.TempDir(), localCredsFile))
		defer tmpCreds.Close()
		if err != nil {
			return nil, err
		}
		logger.Infof("Copying credentials for %v", strings.ToLower(exp.Provider))
		credsPth := filepath.Join(os.Getenv("HOME"), ".ooni", "vpn", exp.Provider+".txt")
		creds, err := os.Open(credsPth)
		if err != nil {
			return nil, err
		}
		defer creds.Close()
		_, err = io.Copy(tmpCreds, creds)
		if err != nil {
			return nil, err
		}

	}

	t := template.New("openvpnConfig")
	t, err := t.Parse(vpnConfigTemplate)
	if err != nil {
		return nil, err
	}
	buf := &bytes.Buffer{}
	err = t.Execute(buf, exp)

	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", "vpn-")
	if err != nil {
		return nil, err
	}
	m.tmpConfigFile = tmp.Name()
	tmp.Write(buf.Bytes())
	opt, err := vpn.NewOptionsFromFilePath(tmp.Name())
	if err != nil {
		return nil, err
	}
	logger.Infof("Using Config File: %s", tmp.Name())

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
		MiniVPNVersion: getMiniVPNVersion(),
		BootstrapTime:  0,
		Failure:        nil,
		Error:          nil,
		Stages:         []Stage{},
		Requests:       []*measurex.ArchivalHTTPRoundTripEvent{},
	}
	sess.Logger().Info("openvpn: bootstrapping openvpn connection")
	defer func() {
		out <- tk
	}()

	vpnEventChan := make(chan uint16, 100)
	m.tunnel.EventListener = vpnEventChan

	zeroTime := time.Now()

	go func() {
		for {
			select {
			case stage := <-vpnEventChan:
				st := newStage(stage, time.Now().Sub(zeroTime))
				tk.Stages = append(tk.Stages, st)
			}
		}
	}()

	index := int64(1)
	trace := measurexlite.NewTrace(index, zeroTime)

	if tk.Transport == "tcp" {
		m.traceDialTCP(ctx, sess, trace, index, tk)
	} else {
		m.dialUDP(ctx, sess, trace, index, tk)
	}

	tk.BootstrapTime = time.Now().Sub(zeroTime).Seconds()
	sess.Logger().Info("openvpn: bootstrapping done")
}

func (m *Measurer) traceDialTCP(ctx context.Context, sess model.ExperimentSession, trace *measurexlite.Trace, index int64, tk *TestKeys) {
	ol := measurexlite.NewOperationLogger(sess.Logger(), "OpenVPN Dial #%d %s", index, tk.Remote)
	dialer := trace.NewDialerWithoutResolver(sess.Logger())
	m.tunnel.Dialer = dialer
	err := m.tunnel.Start(ctx)
	tk.DialConnect = <-trace.TCPConnect
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		tk.Error = &bootstrapError
		sess.Logger().Info("openvpn: bootstrapping failed")
		return
	}
	ol.Stop(err)
}

func (m *Measurer) dialUDP(ctx context.Context, sess model.ExperimentSession, trace *measurexlite.Trace, index int64, tk *TestKeys) {
	err := m.tunnel.Start(ctx)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		tk.Error = &bootstrapError
		sess.Logger().Info("openvpn: bootstrapping failed")
	}
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
			Rtt: toMs(r.Rtt),
			TTL: r.TTL,
		})
	}
	pingStats := &PingResult{
		Target:      target,
		PacketsRecv: st.PacketsRecv,
		PacketsSent: st.PacketsSent,
		Sequence:    replies,
		MinRtt:      toMs(st.MinRtt),
		MaxRtt:      toMs(st.MaxRtt),
		AvgRtt:      toMs(st.AvgRtt),
		StdRtt:      toMs(st.StdDevRtt),
	}
	return pingStats
}

// toMs converts time.Duration to a float64 number representing milliseconds
// with fixed precision (3 decimal places).
func toMs(t time.Duration) float64 {
	return math.Round(t.Seconds()*1e6) / 1e3
}
