// Package openvpn contains the openvpn experiment. This experiment
// measures the bootstrapping of an OpenVPN connection against a given remote,
// a series of ICMP pings, and a series of url page fetches through the tunnel.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-039-openvpn.md
package openvpn

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/ooni/minivpn/vpn"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	// testName is the name of this experiment
	testName = "openvpn"

	// testVersion is the openvpn experiment version.
	testVersion = "0.0.19"

	// pingCount tells how many icmp echo requests to send.
	pingCount = 10

	// sucessLossThreshold will mark icmp pings as successful if loss is below this number.
	sucessLossThreshold = 0.5

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

	// speedTestFiles
	file4kb   = "https://raw.githubusercontent.com/ooni/probe-cli/master/Readme.md"
	file100kb = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_100k.pdf"
	file500kb = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_500k.pdf"
	file1mb   = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_1MB.pdf"
	file10mb  = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_10MB.pdf"
	file20mb  = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_20MB.pdf"
	file50mb  = "https://raw.githubusercontent.com/ainghazal/vpn-test-lists/main/dummy/file_50MB.pdf"
)

var (
	localCredsFile = "ooni-vpn-creds"
)

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

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func registerExtensions(m *model.Measurement) {
	model.ArchivalExtHTTP.AddTo(m)
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

	registerExtensions(measurement)

	tunnel, err := m.setup(experiment, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of
		// time, we probably should log something more specific.
		return err
	}
	m.tunnel = tunnel

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

	var count int
	countFromConfig, err := strconv.Atoi(m.config.PingCount)
	if err == nil {
		count = int(countFromConfig)
	} else {
		fmt.Println("error", err)
		count = pingCount
	}

	// TODO(ainghazal): for the sake of reducing experimental bias, we
	// should randomize the order of the following function calls. But that
	// is going to make parsing the data a bit harder, unless we convene on
	// a given idx. We could explore something like:
	// {'label': 'google.dns', 'target': '8.8.8.8', 'order': 1}
	// TODO(ainghazal): another option pointed out by sbasso is to use a
	// different gvisor socket and then do n pings concurrently. this
	// probably will help with the situation in which packets arrive too
	// late and then they're not arriving from the expected src addr.
	// On the other hand, we should probably compensate for the fact that
	// the tests will take 3x less time to complete (in case the duration
	// of our test was enough to catch delayed firewall response).

	sendBlockingPing(wg, m.tunnel, pingTarget, count, tk)
	sendBlockingPing(wg, m.tunnel, remoteVPNGateway, count, tk)
	sendBlockingPing(wg, m.tunnel, pingTargetNZ, count, tk)

	// we now look at the first two pings to see if we can mark those as
	// "usable" (but see note above about the wish to randomize these icmp
	// pings)
	wantedICMP := 2
	goodICMP := 0
	for _, p := range tk.Pings[:wantedICMP] {
		if p.PacketsSent == 0 {
			break
		}
		loss := 1 - float32(p.PacketsRecv)/float32(p.PacketsSent)
		if loss < sucessLossThreshold {
			goodICMP += 1
		}
	}
	if goodICMP == wantedICMP {
		tk.SuccessICMP = true
	}

	//
	// 2. urlgrab
	//

	sess.Logger().Infof("openvpn: urlgrab stage")

	speedTestTarget := ""
	switch m.config.WithSpeedTest {
	case "4kb":
		speedTestTarget = file4kb
	case "100kb":
		speedTestTarget = file100kb
	case "500kb":
		speedTestTarget = file500kb
	case "1mb":
		speedTestTarget = file1mb
	case "10mb":
		speedTestTarget = file10mb
	case "20mb":
		speedTestTarget = file20mb
	case "50mb":
		speedTestTarget = file50mb
	default:
	}

	doSpeedTest := false
	if speedTestTarget != "" {
		doSpeedTest = true
	}

	targetURLs := []string{urlGrabURI}
	if !doSpeedTest {

		if len(m.config.URLs) != 0 {
			urls := strings.Split(m.config.URLs, ",")
			targetURLs = append(targetURLs, urls...)
		}

		urlgetterConfig := urlgetter.Config{
			Dialer: vpn.NewTunDialer(m.tunnel),
		}

		// TODO this assumes small web pages
		const maxURLGrabTime = 30 * time.Second
		for _, uri := range targetURLs {
			wg.Add(1)
			go func() {
				defer wg.Done()

				m.tunnel.SetReadDeadline(time.Now().Add(time.Second * 60))

				g := urlgetter.Getter{
					Config:  urlgetterConfig,
					Session: sess,
					Target:  uri,
				}
				ctx, cancel := context.WithTimeout(context.Background(), maxURLGrabTime)
				defer cancel()
				urlgetTk, _ := g.Get(ctx)
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
	}

	if doSpeedTest {
		tk.SpeedTest = []*SpeedTest{}
		sess.Logger().Infof("openvpn: speed test")
		sess.Logger().Infof("openvpn: retrieving %s", speedTestTarget)
		// TODO it'd be good to give some feedback in here, we
		// can calculate the progress % for instance.

		// first time, via the direct connection
		wg.Add(1)
		go func() {
			defer wg.Done()
			// TODO(ainghazal): there is probably a better way to do this now,
			// but this is the simplest way I managed to find to pass a custom
			// snapshotSize.
			mx := measurex.NewMeasurerWithDefaultSettings()
			mx.Begin = time.Now()
			const snapshotsize = 1 << 28
			mx.HTTPMaxBodySnapshotSize = snapshotsize
			const timeout = 120 * time.Second
			speedTestResp, err := mx.EasyHTTPRoundTripGET(ctx, timeout, speedTestTarget)
			st := &SpeedTest{
				IsVPN:   false,
				Failure: err,
				Failed:  err != nil,
				File:    speedTestTarget,
			}
			if len(speedTestResp.Requests) > 0 {
				req := speedTestResp.Requests[0]
				st.T0 = req.Started
				st.T = req.Finished
				st.BodyLength = req.Response.BodyLength
			}
			tk.SpeedTest = append(tk.SpeedTest, st)
		}()
		wg.Wait()

		// second time, via the tunnel
		wg.Add(1)
		go func() {
			defer wg.Done()
			// TODO(ainghazal): there is probably a better way to do this now,
			// but this is the simplest way I managed to find to pass a custom
			// snapshotSize.
			mx := measurex.NewMeasurerWithDefaultSettings()
			mx.HTTPClient = &http.Client{
				Transport: &http.Transport{
					Dial: vpn.NewTunDialer(m.tunnel).Dial,
				},
			}
			mx.Begin = time.Now()
			const snapshotsize = 1 << 28
			mx.HTTPMaxBodySnapshotSize = snapshotsize
			const timeout = 120 * time.Second
			speedTestResp, err := mx.EasyHTTPRoundTripGET(ctx, timeout, speedTestTarget)
			st := &SpeedTest{
				IsVPN:   true,
				Failure: err,
				Failed:  err != nil,
				File:    speedTestTarget,
			}
			if len(speedTestResp.Requests) > 0 {
				req := speedTestResp.Requests[0]
				st.T0 = req.Started
				st.T = req.Finished
				st.BodyLength = req.Response.BodyLength
			}
			tk.SpeedTest = append(tk.SpeedTest, st)
		}()
		wg.Wait()
	}

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
	exp.Config.Obfuscation = m.config.Obfuscation

	// we don't want to store the certificates used to test the obfs4 bridge
	exp.Config.ProxyURI = m.config.SafeProxyURI

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

		// TODO retrieve OONI_HOME in a better way
		homedir, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		credsPth := filepath.Join(homedir, ".miniooni", "vpn", exp.Provider+".txt")
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

	obfuscation := experiment.Config.Obfuscation
	var remote string

	switch obfuscation {
	case "obfs4":
		u, _ := url.Parse(experiment.Config.ProxyURI)
		remote = u.Host
	default:
		remote = net.JoinHostPort(m.vpnOptions.Remote, m.vpnOptions.Port)
	}

	tk := NewTestKeys()
	tk.Provider = experiment.Provider
	tk.Transport = protoToString(m.vpnOptions.Proto)
	tk.Remote = remote
	tk.Obfuscation = experiment.Config.Obfuscation

	sess.Logger().Info("openvpn: bootstrapping openvpn connection")
	defer func() {
		out <- tk
	}()

	vpnEventChan := make(chan uint8, 100)
	m.tunnel.EventListener = vpnEventChan

	zeroTime := time.Now()

	wg := &sync.WaitGroup{}
	wg.Add(1)

	go func(wg *sync.WaitGroup) {
		for {
			select {
			case evt := <-vpnEventChan:
				h := newHandshakeEvent(evt, time.Now().Sub(zeroTime))
				tk.HandshakeEvents = append(tk.HandshakeEvents, h)
				if evt == vpn.EventHandshakeDone {
					wg.Done()
				}
			}
		}
	}(wg)

	index := int64(0)
	trace := measurexlite.NewTrace(index, zeroTime)

	if tk.Transport == "tcp" {
		m.traceDialTCP(ctx, sess, trace, index, tk, wg)
	} else {
		m.dialUDP(ctx, sess, trace, index, tk, wg)
	}

	wg.Wait()
	tk.BootstrapTime = time.Now().Sub(zeroTime).Seconds()
	if len(tk.HandshakeEvents) != 0 {
		max := uint8(0)
		for _, e := range tk.HandshakeEvents {
			if e.TransactionID > max {
				max = e.TransactionID
			}
		}
		tk.LastHandshakeTransactionID = uint8(max)
		switch max {
		case vpn.EventHandshakeDone:
			tk.SuccessHandshake = true
		default:
			tk.SuccessHandshake = false
		}
	}
}

func (m *Measurer) traceDialTCP(ctx context.Context, sess model.ExperimentSession, trace *measurexlite.Trace, index int64, tk *TestKeys, wg *sync.WaitGroup) {
	ol := measurexlite.NewOperationLogger(sess.Logger(), "OpenVPN Dial #%d %s", index, tk.Remote)
	dialer := trace.NewDialerWithoutResolver(sess.Logger())
	m.tunnel.Dialer = dialer
	err := m.tunnel.Start(ctx)
	tk.TCPConnect = trace.FirstTCPConnectOrNil()
	ol.Stop(err)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		sess.Logger().Info("openvpn: bootstrapping failed")
		wg.Done() // do not wait for handshake to be completed
		return
	}
	sess.Logger().Info("openvpn: bootstrapping done")
}

func (m *Measurer) dialUDP(ctx context.Context, sess model.ExperimentSession, trace *measurexlite.Trace, index int64, tk *TestKeys, wg *sync.WaitGroup) {
	err := m.tunnel.Start(ctx)
	if err != nil {
		tk.Failure = tracex.NewFailure(err)
		sess.Logger().Info("openvpn: bootstrapping failed")
		wg.Done() // do not wait for handshake to be completed
		return
	}
	sess.Logger().Info("openvpn: bootstrapping done")
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
