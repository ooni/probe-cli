// Package simplequicping is the experimental simplequicping experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-034-simplequicping.md.
package simplequicping

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
	"github.com/quic-go/quic-go"
)

const (
	testName    = "simplequicping"
	testVersion = "0.2.1"
)

// Config contains the experiment configuration.
type Config struct {
	// ALPN allows to specify which ALPN or ALPNs to send.
	ALPN string `json:"alpn,omitempty" ooni:"space separated list of ALPNs to use"`

	// Delay is the delay between each repetition (in milliseconds).
	Delay int64 `json:"delay,omitempty" ooni:"number of milliseconds to wait before sending each ping"`

	// Repetitions is the number of repetitions for each ping.
	Repetitions int64 `json:"repetitions,omitempty" ooni:"number of times to repeat the measurement"`

	// SNI is the SNI value to use.
	SNI string `json:"sni,omitempty" ooni:"the SNI value to use"`
}

func (c *Config) alpn() string {
	if c.ALPN != "" {
		return c.ALPN
	}
	return "h3"
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
	QUICHandshake *model.ArchivalTLSOrQUICHandshakeResult `json:"quic_handshake"`
}

// Measurer performs the measurement.
type Measurer struct{}

// ExperimentName implements ExperimentMeasurer.ExperiExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType

	// errNoInputProvided indicates you didn't provide any input
	errNoInputProvided = errors.New("not input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be quichandshake")

	// errMissingPort indicates that there is no port.
	errMissingPort = errors.New("the URL must include a port")
)

// Run implements ExperimentMeasurer.Run.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// obtain the richer-input target
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*Target)
	if !ok {
		return ErrInvalidInputType
	}
	config, input := target.Config, target.URL

	if input == "" {
		return errNoInputProvided
	}
	parsed, err := url.Parse(input)
	if err != nil {
		return fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "quichandshake" {
		return errInvalidScheme
	}
	if parsed.Port() == "" {
		return errMissingPort
	}
	tk := new(TestKeys)
	measurement.TestKeys = tk
	out := make(chan *SinglePing)
	go m.simpleQUICPingLoop(ctx, config, measurement.MeasurementStartTimeSaved, sess.Logger(), parsed.Host, out)
	for len(tk.Pings) < int(config.repetitions()) {
		tk.Pings = append(tk.Pings, <-out)
	}
	return nil // return nil so we always submit the measurement
}

// simpleQUICPingLoop sends all the ping requests and emits the results onto the out channel.
func (m *Measurer) simpleQUICPingLoop(ctx context.Context, config *Config, zeroTime time.Time,
	logger model.Logger, address string, out chan<- *SinglePing) {
	ticker := time.NewTicker(config.delay())
	defer ticker.Stop()
	for i := int64(0); i < config.repetitions(); i++ {
		go m.simpleQUICPingAsync(ctx, config, i, zeroTime, logger, address, out)
		<-ticker.C
	}
}

// simpleQUICPingAsync performs a QUIC ping and emits the result onto the out channel.
func (m *Measurer) simpleQUICPingAsync(ctx context.Context, config *Config, index int64,
	zeroTime time.Time, logger model.Logger, address string, out chan<- *SinglePing) {
	out <- m.quicHandshake(ctx, config, index, zeroTime, logger, address)
}

// quicHandshake performs a QUIC handshake and returns the results of these operations to the caller.
func (m *Measurer) quicHandshake(ctx context.Context, config *Config, index int64,
	zeroTime time.Time, logger model.Logger, address string) *SinglePing {
	// TODO(bassosimone): make the timeout user-configurable
	ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	sp := &SinglePing{
		NetworkEvents: []*model.ArchivalNetworkEvent{},
		QUICHandshake: nil,
	}
	sni := config.sni(address)
	alpn := strings.Split(config.alpn(), " ")
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := logx.NewOperationLogger(logger, "SimpleQUICPing #%d %s %s %v", index, address, sni, alpn)
	netx := &netxlite.Netx{}
	listener := netx.NewUDPListener()
	dialer := trace.NewQUICDialerWithoutResolver(listener, logger)
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	tlsConfig := &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		NextProtos: alpn,
		RootCAs:    nil,
		ServerName: sni,
	}
	quicEarlyConn, err := dialer.DialContext(ctx, address, tlsConfig, &quic.Config{})
	defer measurexlite.MaybeCloseQUICConn(quicEarlyConn)
	ol.Stop(err)
	sp.QUICHandshake = trace.FirstQUICHandshakeOrNil() // record the first handshake from the buffer
	sp.NetworkEvents = append(sp.NetworkEvents, trace.NetworkEvents()...)
	return sp
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}
