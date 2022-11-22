package bittorrent

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/anacrolix/torrent"
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
	// golint is stupid and does not let us end erorr with ":"
	errInvalidScheme = errors.New("scheme must be magnet")
)

const (
	testName    = "bittorrent"
	testVersion = "0.0.1"
)

// Config contains the experiment config.
type Config struct{}

type runtimeConfig struct {
	magnet string
}

func config(input model.MeasurementTarget) (*runtimeConfig, error) {
	if input == "" {
		return nil, errNoInputProvided
	}

	parsed, err := url.Parse(string(input))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "magnet" {
		return nil, errInvalidScheme
	}

	validConfig := runtimeConfig{
		magnet: string(input),
	}

	return &validConfig, nil
}

// TestKeys contains the experiment results
type TestKeys struct {
	// DNS queries when resolving trackers
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`
	// Indicates any kind of failure
	Failure string `json:"failure"`
	// The total number of peers contacted about the requested magnet
	PeersNum int `json:"peers_num"`
	// The complete list of peers contacted
	Peers []string `json:"peers"`
	// The total number of bytes received by the client
	TotalBytesRead int64 `json:"total_bytes_received"`
	// The total number of bad pieces (failed verification) received by the client
	TotalBadPieces int64 `json:"total_bad_pieces"`
}

func (tk *TestKeys) failure(err error) {
	tk.Failure = *tracex.NewFailure(err)
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

func torrentStats(torrent *torrent.Torrent, client *torrent.Client, tk *TestKeys) {
	stats := torrent.Stats()
	tk.PeersNum = len(tk.Peers)
	tk.TotalBytesRead = stats.ConnStats.BytesRead.Int64()
	tk.TotalBadPieces = stats.ConnStats.PiecesDirtiedBad.Int64()
}

func timeoutStats(torrent *torrent.Torrent, client *torrent.Client, tk *TestKeys) {
	torrentStats(torrent, client, tk)
	tk.Failure = "download_timeout"
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

	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	tmpdir, err := os.MkdirTemp("", "ooni")
	if err != nil {
		log.Warnf(*tracex.NewFailure(err))
		return nil
	}
	log.Infof("Using temporary directory %s", tmpdir)
	defer os.RemoveAll(tmpdir)

	conf := torrent.NewDefaultClientConfig()
	conf.DataDir = tmpdir
	conf.NoUpload = true

	// Lookup tracker IPs via ooni utils
	conf.LookupTrackerIp = func(u *url.URL) ([]net.IP, error) {

		log.Infof("Resolving DNS for %s", u.Hostname())
		resolvedAddrs, err := resolver.LookupHost(ctx, u.Hostname())
		addrs := []net.IP{}
		if err != nil {
			return addrs, nil
		}
		log.Infof("Finished DNS for %s: %v", u.Hostname(), resolvedAddrs)
		for _, addr := range resolvedAddrs {
			addrs = append(addrs, net.ParseIP(addr))
		}
		tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
		return addrs, err
	}

	// We want to test Bittorrent connectivity, not HTTPS/websockets
	conf.DisableWebtorrent = true
	conf.DisableWebseeds = true

	// Register new peers to the test keys
	clientCallbacks := new(torrent.Callbacks)
	clientCallbacks.NewPeer = append(clientCallbacks.NewPeer,
		func(peer *torrent.Peer) {
			log.Debugf("Found new peer: %s", peer.RemoteAddr.String())
			tk.Peers = append(tk.Peers, peer.RemoteAddr.String())
		},
	)
	conf.Callbacks = *clientCallbacks

	client, err := torrent.NewClient(conf)
	if err != nil {
		log.Warnf(*tracex.NewFailure(err))
		return nil
	}
	defer client.Close()

	torrent, err := client.AddMagnet(config.magnet)
	if err != nil {
		log.Warnf(*tracex.NewFailure(err))
		return nil
	}

	select {
	case <-ctx.Done():
		tk.Failure = "metainfo_timeout"
		return nil
	case <-torrent.GotInfo():
	}

	torrent.DownloadAll()

	// Setup a new chan to know when the torrent is finished... allows to apply timeout
	finished := make(chan bool)

	go func() {
		client.WaitAll()
		finished <- true
	}()
	select {
	case <-ctx.Done():
		timeoutStats(torrent, client, tk)
	case <-finished:
		torrentStats(torrent, client, tk)
	}
	return nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{Config: config}
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
