package dht

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net/url"
	"time"

	"github.com/anacrolix/dht/v2"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

var (
	// errNoInputProvided indicates no input was passed
	errNoInputProvided = errors.New("no input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be dht://")

	// errMissingPort indicates that no port was provided
	errMissingPort = errors.New("no port was provided but dht:// requires explicit port")
)

const (
	testName    = "dht"
	testVersion = "0.0.1"
)

// Config contains the experiment config.
type Config struct{}

type runtimeConfig struct {
	// nodeaddr IP or domain name
	dhtnode  string
	port     string
	infohash string
}

func config(input model.MeasurementTarget) (*runtimeConfig, error) {
	// Bittorrent v2 hybrid test torrent: https://blog.libtorrent.org/2020/09/bittorrent-v2/
	// Has good chances of being seeded years from now
	hash := "631a31dd0a46257d5078c0dee4e66e26f73e42ac"

	if input == "" {
		return nil, errNoInputProvided
	}

	// TODO: static input from defaultDHTBoostrapNodes()
	// input == "" triggers runtime error from the experiment runner
	if input == "DUMMY" {
		// No requested DHT bootstrap node, let the DHT library try all it knows
		return &runtimeConfig{
			dhtnode:  "",
			port:     "",
			infohash: hash,
		}, nil
	}

	parsed, err := url.Parse(string(input))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "dht" {
		return nil, errInvalidScheme
	}

	if parsed.Port() == "" {
		// Port is mandatory because DHT bootstrap nodes use different ports
		return nil, errMissingPort
	}

	validConfig := runtimeConfig{
		dhtnode:  parsed.Hostname(),
		port:     parsed.Port(),
		infohash: hash,
	}

	return &validConfig, nil
}

// TestKeys contains the experiment results
type TestKeys struct {
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`
	Runs    []*IndividualTestKeys            `json:"runs"`
	// Used for global failure (DNS resolution)
	Failure string `json:"failure"`
}

func (tk *TestKeys) failure(err error) {
	tk.Failure = *tracex.NewFailure(err)
}

func (tk *TestKeys) computeFailure() {
	if tk.Failure != "" {
		return
	}
	for _, itk := range tk.Runs {
		if itk.Failure != "" {
			tk.Failure = itk.Failure
			return
		}
	}
}

// IndividualTestKeys indicate results for a single IP/port combo DHT bootstrap node
// in case the DNS resolves to several IPs, or multiple bootstrap domains were used
type IndividualTestKeys struct {
	// Logger, not exported to JSON
	logger model.Logger

	// List of IP/port combos tried to boostrap DHT
	BootstrapNodes []string `json:"bootstrap_nodes"`
	// Number of DHT bootsrap nodes
	BootstrapNum int `json:"bootstrap_num"`
	// Number of DHT peers contacted
	PeersTriedNum uint32 `json:"peers_tried_num"`
	// Number of DHT peers who answered
	PeersRespondedNum uint32 `json:"peers_responded_num"`
	// Number of DHT peers found for specific requested infohash
	InfohashPeersNum int `json:"infohash_peers_num"`
	// Actual DHT peers found for requested infohash
	InfohashPeers []string `json:"infohash_peers"`
	// Individual failure aborting the test run for this address/port combo
	Failure string `json:"failure"`
}

func (itk *IndividualTestKeys) error(err error) {
	itk.Failure = *tracex.NewFailure(err)
	itk.logger.Warn(itk.Failure)
}

func newITK(tk *TestKeys, log model.Logger) *IndividualTestKeys {
	itk := new(IndividualTestKeys)
	itk.logger = log
	tk.Runs = append(tk.Runs, itk)
	return itk
}

// Measurer performs the measurement.
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config

	// Getter is an optional getter to be used for testing.
	Getter urlgetter.MultiGetter
}

// ExperimentName implements ExperimentMeasurer.ExperimentName
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

func defaultDHTBoostrapNodes() []string {
	return []string{
		"router.utorrent.com:6881",
		"router.bittorrent.com:6881",
		"dht.transmissionbt.com:6881",
		"dht.aelitis.com:6881",
		"router.silotis.us:6881",
		"dht.libtorrent.org:25401",
		"dht.anacrolix.link:42069",
		"router.bittorrent.cloud:42069",
	}
}

// Server starts a DHT server with a list of bootstrap nodes and stores
// failure cases inside a IndividualTestKeys
func Server(bootstrapNodes []string, itk *IndividualTestKeys) (*dht.Server, bool) {
	itk.BootstrapNodes = bootstrapNodes
	itk.BootstrapNum = len(bootstrapNodes)

	// Starting new DHT client
	dhtconf := dht.NewDefaultServerConfig()
	dhtconf.QueryResendDelay = func() time.Duration {
		return 10 * time.Second
	}

	dhtconf.StartingNodes = func() (addrs []dht.Addr, err error) {
		for _, addrport := range bootstrapNodes {
			udpAddr, err := net.ResolveUDPAddr("udp", addrport)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, dht.NewAddr(udpAddr))
		}
		return addrs, nil
	}

	dhtsrv, err := dht.NewServer(dhtconf)
	if err != nil {
		itk.error(err)
		return nil, false
	}
	itk.logger.Infof("Finished starting DHT server with bootstrap nodes: %v", bootstrapNodes)
	return dhtsrv, true
}

func testServer(dht *dht.Server, infohash [20]byte, itk *IndividualTestKeys) bool {
	announce, err := dht.AnnounceTraversal(infohash)
	if err != nil {
		itk.error(err)
		return false
	}
	defer announce.Close()

	counter := 0
	for entry := range announce.Peers {
		counter++
		itk.InfohashPeers = append(itk.InfohashPeers, entry.NodeInfo.Addr.String())
		itk.logger.Debugf("peer %d: %s", counter, entry.NodeInfo.Addr)
	}

	stats := announce.TraversalStats()
	itk.PeersTriedNum = stats.NumAddrsTried
	itk.PeersRespondedNum = stats.NumResponses
	itk.InfohashPeersNum = counter

	if itk.PeersRespondedNum == 0 {
		itk.error(errors.New("No DHT peers were found"))
		return false
	}

	itk.logger.Infof("Tried %d peers obtained from %d bootstrap nodes. Got response from %d. %d have requested infohash.", itk.PeersTriedNum, itk.BootstrapNum, itk.PeersRespondedNum, itk.InfohashPeersNum)

	return true

}

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	//ctx context.Context, sess model.ExperimentSession,
	//measurement *model.Measurement, callbacks model.ExperimentCallbacks,
	//) error {
	sess := args.Session
	measurement := args.Measurement
	log := sess.Logger()
	trace := measurexlite.NewTrace(0, measurement.MeasurementStartTimeSaved)
	resolver := trace.NewStdlibResolver(log)

	config, err := config(measurement.Input)
	if err != nil {
		// Invalid input data, we don't even generate report
		return err
	}

	tk := new(TestKeys)
	measurement.TestKeys = tk

	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	// Turn string infohash into 20-bytes array
	var infohash [20]byte
	copy(infohash[:], config.infohash)

	if config.dhtnode != "" {
		// Specific node provided: resolve it
		log.Infof("Resolving DNS for %s", config.dhtnode)
		resolvedAddrs, err := resolver.LookupHost(ctx, config.dhtnode)
		tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
		if err != nil {
			tk.failure(err)
			return nil
		}
		log.Infof("Finished DNS for %s: %v", config.dhtnode, resolvedAddrs)

		for _, addr := range resolvedAddrs {

			nodeAddrport := net.JoinHostPort(addr, config.port)
			log.Infof("Trying DHT bootstrap node %s", nodeAddrport)
			nodeAddrports := []string{nodeAddrport}

			itk := newITK(tk, log)

			dht, success := Server(nodeAddrports, itk)
			if !success {
				continue
			}

			testServer(dht, infohash, itk)
		}
	} else {
		// Use default DHT bootstrap nodes because none was given by input
		resolvedAddrports := []string{}
		for _, bootstrapDomain := range defaultDHTBoostrapNodes() {
			// Ignore error because we use static input so panic chance is 0
			host, port, _ := net.SplitHostPort(bootstrapDomain)
			log.Infof("Resolving DNS for %s", host)
			resolvedAddrs, err := resolver.LookupHost(ctx, host)
			tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
			if err != nil {
				tk.failure(err)
				return nil
			}
			log.Infof("Finished DNS for %s: %v", host, resolvedAddrs)
			for _, resolvedAddr := range resolvedAddrs {
				resolvedAddrports = append(resolvedAddrports, net.JoinHostPort(resolvedAddr, port))
			}
		}
		log.Infof("Resolved the following bootstrap nodes: %v", resolvedAddrports)

		itk := newITK(tk, log)
		dht, success := Server(resolvedAddrports, itk)
		if success {
			testServer(dht, infohash, itk)
		}
	}

	tk.computeFailure()

	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	_, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	return sk, nil
}
