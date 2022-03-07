// Package wireguard contains the wireguard experiment. This experiment
// measures the bootstrapping of an WireGuard connection against a given remote.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-033-wireguard.md

package wireguard

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net/netip"
	"os"
	"strings"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun/netstack"

	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	icmpTimeoutSeconds = 10
	errBadBootstrap    = "bad_bootstrap"
	errPingError       = "ping_error"
	errBadReplyType    = "ping_bad_reply_type"
	errBadReply        = "ping_bad_reply"
)

type options struct {
	ip       string
	pubKey   string
	privKey  string
	endpoint string
	ns       string
}

func getOptionsFromFile(f string) (*options, error) {
	o := &options{}
	lines, err := getLinesFromFile(f)
	if err != nil {
		return nil, err
	}
	for i, l := range lines {
		if strings.HasPrefix(l, "#") {
			continue
		}
		p := strings.Split(l, "=")
		if len(p) != 2 {
			return nil, fmt.Errorf("wrong line (%d): more than one =", i)
		}
		k, v := p[0], p[1]
		switch k {
		case "ip":
			o.ip = v
		case "public_key":
			o.pubKey = v
		case "private_key":
			o.privKey = v
		case "endpoint":
			o.endpoint = v
		default:
			// do nothing
		}

	}
	o.ns = defaultNameserver
	//fmt.Println("options:")
	//fmt.Println(o)
	return o, nil
}

const (
	// testName is the name of this experiment
	testName = "wireguard"

	// testVersion is the wireguard experiment version.
	testVersion = "0.0.1"

	// pingCount tells how many icmp echo requests to send.
	pingCount = 3

	// pingTarget is the target IP we used for pings.
	pingTarget = "8.8.8.8"

	// defaultNameserver is the dns server using for resolving names inside the wg tunnel.
	defaultNameserver = "8.8.8.8"

	// urlGrabURI is an URI we fetch to check web connectivity and external IP.
	urlGrabURI = "https://api.ipify.org/?format=json"
)

// Config contains the experiment config.
type Config struct {
	ConfigFile string `ooni:"Configuration file for the WireGuard experiment"`
}

type Ping struct {
	RTT float32 `json:"rtt"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// VPNLogs contains the bootstrap logs.
	VPNLogs []string `json:"wg_logs"`

	// Pings is an array of ping stats.
	Pings []Ping `json:"pings"`

	// PingTarget is the target we used for ping
	PingTarget string `json:"ping_target"`
}

// Measurer performs the measurement.
type Measurer struct {
	// config contains the experiment settings.
	config  Config
	options *options
	device  device.Device
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
	config := string(measurement.Input)
	err := m.setup(ctx, config, sess.Logger())
	if err != nil {
		// we cannot setup the experiment
		// TODO this includes if we don't have the correct certificates etc.
		// This means that we need to get the cert material ahead of time.
		return err
	}
	m.registerExtensions(measurement)

	//start := time.Now()
	const maxRuntime = 600 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
	tkch := make(chan *TestKeys)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// TODO pass timeout context
	go m.bootstrap(ctx, sess, tkch)
	for {
		select {
		case tk := <-tkch:
			measurement.TestKeys = tk
			callbacks.OnProgress(1.0, testName+" bootstrap done")
			return nil
		}
		// todo: progress...
	}
}

// setup prepares for running the wireguard experiment. Returns a minivpn dialer on success.
// Returns an error on failure.
func (m *Measurer) setup(ctx context.Context, config string, logger model.Logger) error {
	o, err := getOptionsFromFile(config)
	if err != nil {
		return err
	}
	m.options = o
	return err
}

// bootstrap runs the bootstrap.
func (m *Measurer) bootstrap(ctx context.Context, sess model.ExperimentSession,
	out chan<- *TestKeys) {
	tk := &TestKeys{
		BootstrapTime: 0,
		Failure:       nil,
	}
	sess.Logger().Info("wireguard: bootstrapping experiment")
	defer func() {
		out <- tk
	}()

	s := time.Now()
	tnet, err := doWireguardBootstrap(m.options)
	if err != nil {
		tk.Failure = &errBadBootstrap
		return
	}

	// this bootstrap is only local interface setup, so maybe it does not make sense to
	// include it in measurements.
	tk.BootstrapTime = time.Now().Sub(s).Seconds()

	// TODO separate to a different function
	stats, err := doPings(tnet)
	if err != nil {
		// TODO check different types of error
		tk.Failure = &errPingError
	}
	tk.Pings = stats
	tk.PingTarget = pingTarget

	// TODO capture logs
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

func doWireguardBootstrap(o *options) (*netstack.Net, error) {
	tun, tnet, err := netstack.CreateNetTUN(
		[]netip.Addr{netip.MustParseAddr(o.ip)},
		[]netip.Addr{netip.MustParseAddr(o.ns)},
		1420)
	if err != nil {
		log.Panic(err)
	}
	dev := device.NewDevice(tun, conn.NewDefaultBind(), device.NewLogger(device.LogLevelVerbose, ""))
	dev.IpcSet(`private_key=` + o.privKey + `
public_key=` + o.pubKey + `
endpoint=` + o.endpoint + `
allowed_ip=0.0.0.0/0
`)
	err = dev.Up()
	if err != nil {
		return nil, err
	}
	return tnet, nil

}

func doPings(tnet *netstack.Net) ([]Ping, error) {
	stats := []Ping{}
	for i := 0; i < pingCount; i++ {
		socket, err := tnet.Dial("ping4", pingTarget)
		if err != nil {
			return nil, err
		}
		requestPing := icmp.Echo{
			Seq:  rand.Intn(1 << 16),
			Data: []byte("hello dpi"),
		}
		icmpBytes, _ := (&icmp.Message{Type: ipv4.ICMPTypeEcho, Code: 0, Body: &requestPing}).Marshal(nil)
		socket.SetReadDeadline(time.Now().Add(time.Second * time.Duration(icmpTimeoutSeconds)))
		start := time.Now()
		_, err = socket.Write(icmpBytes)
		if err != nil {
			log.Panic(err)
		}
		n, err := socket.Read(icmpBytes[:])
		if err != nil {
			log.Panic(err)
		}
		replyPacket, err := icmp.ParseMessage(1, icmpBytes[:n])
		if err != nil {
			log.Panic(err)
		}
		replyPing, ok := replyPacket.Body.(*icmp.Echo)
		if !ok {
			log.Printf("invalid reply type: %v\n", replyPacket)
			return stats, fmt.Errorf(errBadReply)
		}
		if !bytes.Equal(replyPing.Data, requestPing.Data) || replyPing.Seq != requestPing.Seq {
			log.Printf("invalid ping reply: %v\n", replyPing)
			return stats, fmt.Errorf(errBadReply)
		}
		rtt := time.Since(start)
		log.Printf("RTT: : %v ms", rtt)
		stats = append(stats, Ping{float32(rtt)})
	}
	return stats, nil
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
