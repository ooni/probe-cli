// Package tlsping is the experimental tlsping experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-033-tlsping.md.
package tlsping

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "tlsping"
	testVersion = "0.2.0"
)

// Config contains the experiment configuration.
type Config struct {
	// ALPN allows to specify which ALPN or ALPNs to send.
	ALPN string `ooni:"space separated list of ALPNs to use"`

	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `ooni:"number of milliseconds to wait before sending each ping"`

	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `ooni:"number of times to repeat the measurement"`

	// SNI is the SNI value to use.
	SNI string `ooni:"the SNI value to use"`
}

func (c *Config) alpn() string {
	if c.ALPN != "" {
		return c.ALPN
	}
	return "h2 http/1.1"
}

func (c *Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return time.Second
}

func (c *Config) repetitions() int64 {
	if c.Repetitions > 0 {
		return c.Repetitions
	}
	return 10
}

func (c *Config) sni(address string) string {
	if c.SNI != "" {
		return c.SNI
	}
	addr, _, err := net.SplitHostPort(address)
	if err != nil {
		return ""
	}
	return addr
}

// TestKeys contains the experiment results.
type TestKeys struct {
	Pings []*SinglePing `json:"pings"`
}

// SinglePing contains the results of a single ping.
type SinglePing struct {
	NetworkEvents []*model.ArchivalNetworkEvent           `json:"network_events"`
	TCPConnect    *model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	TLSHandshake  *model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshake"`
}

// Measurer performs the measurement.
type Measurer struct {
	config Config
}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// errNoInputProvided indicates you didn't provide any input
	errNoInputProvided = errors.New("not input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be tlshandshake")

	// errMissingPort indicates that there is no port.
	errMissingPort = errors.New("the URL must include a port")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(
	ctx context.Context,
	sess model.ExperimentSession,
	measurement *model.Measurement,
	callbacks model.ExperimentCallbacks,
) error {
	if measurement.Input == "" {
		return errNoInputProvided
	}
	parsed, err := url.Parse(string(measurement.Input))
	if err != nil {
		return fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "tlshandshake" {
		return errInvalidScheme
	}
	if parsed.Port() == "" {
		return errMissingPort
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	out := make(chan *SinglePing)
	go m.tlsPingLoop(ctx, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, out)
	for len(tk.Pings) < int(m.config.repetitions()) {
		tk.Pings = append(tk.Pings, <-out)
	}
	return nil // return nil so we always submit the measurement
}

// tlsPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) tlsPingLoop(ctx context.Context, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *SinglePing) {
	ticker := time.NewTicker(m.config.delay())
	defer ticker.Stop()
	for i := int64(0); i < m.config.repetitions(); i++ {
		go m.tlsPingAsync(ctx, i, zeroTime, logger, address, out)
		<-ticker.C
	}
}

// tlsPingAsync performs a TLS ping and emits the result onto the out channel.
func (m *Measurer) tlsPingAsync(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string, out chan<- *SinglePing) {
	out <- m.tlsConnectAndHandshake(ctx, index, zeroTime, logger, address)
}

// tlsConnectAndHandshake performs a TCP connect followed by a TLS handshake
// and returns the results of these operations to the caller.
func (m *Measurer) tlsConnectAndHandshake(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, address string) *SinglePing {
	// TODO(bassosimone): make the timeout user-configurable
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	sp := &SinglePing{
		NetworkEvents: []*model.ArchivalNetworkEvent{},
		TCPConnect:    nil,
		TLSHandshake:  nil,
	}
	trace := measurexlite.NewTrace(index, zeroTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	alpn := strings.Split(m.config.alpn(), " ")
	sni := m.config.sni(address)
	ol := measurexlite.NewOperationLogger(logger, "TLSPing #%d %s %s %v", index, address, sni, alpn)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	sp.TCPConnect = trace.FirstTCPConnect() // record the first connect from the buffer
	if err != nil {
		ol.Stop(err)
		return sp
	}
	defer conn.Close()
	conn = trace.WrapNetConn(conn)
	thx := trace.NewTLSHandshakerStdlib(logger)
	config := &tls.Config{
		NextProtos: alpn,
		RootCAs:    netxlite.NewDefaultCertPool(),
		ServerName: sni,
	}
	_, _, err = thx.Handshake(ctx, conn, config)
	ol.Stop(err)
	sp.TLSHandshake = trace.FirstTLSHandshake() // record the first handshake from the buffer
	sp.NetworkEvents = trace.NetworkEvents()
	return sp
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
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
	return SummaryKeys{IsAnomaly: false}, nil
}
