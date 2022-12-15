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
type Config struct{
	DisableDHTSecurity bool
}

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
		dhtnode:  fmt.Sprintf("[%s]", parsed.Hostname()),
		port:     parsed.Port(),
		infohash: hash,
	}

	return &validConfig, nil
}

// TestKeys contains the experiment results
type TestKeys struct {
	// DNS queries for the requested boostrap nodes
	// TODO: move to individual test keys? https://github.com/ooni/probe-cli/pull/986#issuecomment-1327659474
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`
	Runs    []*IndividualTestKeys            `json:"runs"`
	// Used for global failure (DNS resolution)
	Failure string `json:"failure"`
}

// DHTRunner takes care of sequential DHT tests
type DHTRunner struct {
	// Only used for building multiple runs from a global bootstrap nodes list
	tk                *TestKeys
	// Where to store run results
	itk				  *IndividualTestKeys
	ctx				  context.Context
	trace             *measurexlite.Trace
	logger            model.Logger
	BootstrapNodes    []string
	resolvedNodes     []string
	disableDHTSecurity bool
}

func (d *DHTRunner) error(msg string) {
	d.itk.Failure = msg
	d.globalError(msg)
}

func (d *DHTRunner) globalError(msg string) {
	d.tk.Failure = msg
}

// resolve takes the current list of bootstrap nodes (potentially domain names)
// and resolves them to actual IP addresses and ports for further use in runs
func (d *DHTRunner) resolve() bool {
	// If no BootstrapNodes were passed, use default list
	if len(d.BootstrapNodes) == 0 {
		d.BootstrapNodes = defaultDHTBoostrapNodes()
	}

	resolver := d.trace.NewStdlibResolver(d.logger)
	resolveCounter := 0
	successCounter := 0

	for _, node := range(d.BootstrapNodes) {
		resolveCounter++

		host, port, err := net.SplitHostPort(node)
		if err != nil {
			// Provided bootstrap node is not valid host:port, abort
			d.globalError(*tracex.NewFailure(err))
			return false
		}

		d.logger.Infof("Starting DNS for '%s'", host)
		addrs, err := resolver.LookupHost(d.ctx, host)
		d.tk.Queries = append(d.tk.Queries, d.trace.DNSLookupsFromRoundTrip()...)
		if err != nil {
			// Failed to resolve host, don't abort
			d.logger.Warn(*tracex.NewFailure(err))
			continue
		}
		successCounter++
		d.logger.Infof("Finished DNS for '%s' : %v", host, addrs)

		// Append individual IP/port to resolvedNodes
		for _, ip := range(addrs)  {
			d.resolvedNodes = append(d.resolvedNodes, net.JoinHostPort(ip, port))
		}
	}

	if resolveCounter != successCounter {
		d.logger.Warn("Some DNS resolutions failed (see errors above)")
	}

	// If all resolutions failed, return error
	if successCounter == 0 {
		d.globalError("All provided bootstrap nodes failed to resolve")
		return false
	}

	return true
}

// runSeparate tests all resolved IP/port combos as separate DHT runs
// Takes a refTime to create a new trace for each run
func (d *DHTRunner) runSeparate(refTime time.Time, infohash [20]byte) bool {
	oneOrMoreFailed := false

	if ! d.resolve() {
		return false
	}

	for _, node := range(d.resolvedNodes) {
		// Start a new 
		subRunner := &DHTRunner{
			tk:        		d.tk,
			itk:			d.newRun(),
			ctx:			d.ctx,
			// TODO: populate NewTrace time
			trace:          measurexlite.NewTrace(0, refTime),
			logger:         d.logger,
			BootstrapNodes: []string{},
			resolvedNodes:  []string{node},
			disableDHTSecurity: d.disableDHTSecurity,
		}

		// Ignore individual errors. They are stored as failure but we want to keep iterating
		if ! subRunner.bootstrap(infohash) {
			oneOrMoreFailed = true
		}
	}

	return oneOrMoreFailed
}

func (d *DHTRunner) newRun() *IndividualTestKeys {
	itk := new(IndividualTestKeys)
	d.tk.Runs = append(d.tk.Runs, itk)
	return itk
}

func (d *DHTRunner) run(infohash [20]byte) bool {
	if ! d.resolve() {
		return false
	}

	d.itk = d.newRun()
	return d.bootstrap(infohash)
}

func (d *DHTRunner) bootstrap(infohash [20]byte) bool {
	d.itk.BootstrapNodes = d.resolvedNodes
	d.itk.BootstrapNum = len(d.resolvedNodes)

	// Starting new DHT client
	d.logger.Infof("Starting DHT server for the following bootstrap nodes: %v", d.resolvedNodes)
	dhtconf := dht.NewDefaultServerConfig()
	dhtconf.QueryResendDelay = func() time.Duration {
		return 10 * time.Second
	}

	dhtconf.StartingNodes = func() (addrs []dht.Addr, err error) {
		for _, addrport := range d.resolvedNodes {
			udpAddr, err := net.ResolveUDPAddr("udp", addrport)
			if err != nil {
				return nil, err
			}
			addrs = append(addrs, dht.NewAddr(udpAddr))
		}
		return addrs, nil
	}

	// Disable DHT security for local tests
	if d.disableDHTSecurity {
		dhtconf.NoSecurity = true
	}

	dhtsrv, err := dht.NewServer(dhtconf)
	if err != nil {
		d.itk.error(err)
		return false
	}
	d.logger.Infof("Finished starting DHT server. Starting announce for %s", string(infohash[:]))

	announce, err := dhtsrv.AnnounceTraversal(infohash)
	if err != nil {
		d.itk.error(err)
		return false
	}
	defer announce.Close()

	counter := 0
	for entry := range announce.Peers {
		counter++
		d.itk.InfohashPeers = append(d.itk.InfohashPeers, entry.NodeInfo.Addr.String())
		d.logger.Debugf("peer %d: %s", counter, entry.NodeInfo.Addr)
	}

	stats := announce.TraversalStats()
	d.itk.PeersTriedNum = stats.NumAddrsTried
	d.itk.PeersRespondedNum = stats.NumResponses
	d.itk.InfohashPeersNum = counter

	if d.itk.PeersRespondedNum == 0 {
		d.error("No DHT peers were found")
		return false
	}

	d.logger.Infof("Tried %d peers obtained from %d bootstrap nodes. Got response from %d. %d have requested infohash.", d.itk.PeersTriedNum, d.itk.BootstrapNum, d.itk.PeersRespondedNum, d.itk.InfohashPeersNum)

	return true
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

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	//ctx context.Context, sess model.ExperimentSession,
	//measurement *model.Measurement, callbacks model.ExperimentCallbacks,
	//) error {
	sess := args.Session
	measurement := args.Measurement
	log := sess.Logger()
	trace := measurexlite.NewTrace(0, measurement.MeasurementStartTimeSaved)

	//resolver := trace.NewStdlibResolver(log)
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

	runner := &DHTRunner{
		tk:        		tk,
		itk:			nil,
		ctx:			ctx,
		trace:          trace,
		logger:         log,
		// TODO: default to boostrap
		BootstrapNodes: []string{},
		resolvedNodes:  []string{},
		disableDHTSecurity: m.Config.DisableDHTSecurity,
	}

	if config.dhtnode != "" {
		// We only want to try the specified node, using all IPs as separate runs
		runner.BootstrapNodes = []string{fmt.Sprintf("%s:%s", config.dhtnode, config.port)}
		runner.runSeparate(measurement.MeasurementStartTimeSaved, infohash)
	} else {
		// We want to try all default nodes in a single run
		runner.run(infohash)
	}

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
