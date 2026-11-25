package imap

import (
	"bufio"
	"context"
	"crypto/tls"
	"fmt"
	"github.com/pkg/errors"
	"net/url"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/tcprunner"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

var (
	// errNoInputProvided indicates you didn't provide any input
	errNoInputProvided = errors.New("not input provided")

	// errInputIsNotAnURL indicates that input is not an URL
	errInputIsNotAnURL = errors.New("input is not an URL")

	// errInvalidScheme indicates that the scheme is invalid
	errInvalidScheme = errors.New("scheme must be imap(s)")
)

const (
	testName    = "imap"
	testVersion = "0.0.1"
)

// Config contains the experiment config.
type Config struct{}

type runtimeConfig struct {
	host      string
	port      string
	forcedTLS bool
	noopCount uint8
}

func config(input model.MeasurementTarget) (*runtimeConfig, error) {
	if input == "" {
		// TODO: static input data (eg. gmail/riseup..)
		return nil, errNoInputProvided
	}

	parsed, err := url.Parse(string(input))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", errInputIsNotAnURL, err.Error())
	}
	if parsed.Scheme != "imap" && parsed.Scheme != "imaps" {
		return nil, errInvalidScheme
	}

	port := ""

	if parsed.Port() == "" {
		// Default ports for StartTLS and forced TLS respectively
		if parsed.Scheme == "imap" {
			port = "143"
		} else {
			port = "993"
		}
	} else {
		// Valid port is checked by URL parsing
		port = parsed.Port()
	}

	validConfig := runtimeConfig{
		host:      parsed.Hostname(),
		forcedTLS: parsed.Scheme == "imaps",
		port:      port,
		noopCount: 10,
	}

	return &validConfig, nil
}

// TestKeys contains the experiment results for an entire domain host
type TestKeys struct {
	Host    string                           `json:"hostname"`
	Queries []*model.ArchivalDNSLookupResult `json:"queries"`
	// Individual IP/port results
	Runs []*IndividualTestKeys `json:"runs"`
	// Used for global failure (DNS resolution)
	Failure string `json:"failure"`
}

func newTestKeys(host string) *TestKeys {
	tk := new(TestKeys)
	tk.Host = host
	return tk
}

// Hostname TCPRunnerModel
func (tk *TestKeys) Hostname(host string) {
	tk.Host = host
}

// DNSResults TCPRunnerModel
func (tk *TestKeys) DNSResults(res []*model.ArchivalDNSLookupResult) {
	// TODO: not sure if we are passed the overall trace results and should overwrite key, or just append
	tk.Queries = append(tk.Queries, res...)
}

// Failed TCPRunnerModel
func (tk *TestKeys) Failed(msg string) {
	tk.Failure = msg
}

// NewRun TCPRunnerModel
func (tk *TestKeys) NewRun(addr string, port string) tcprunner.TCPSessionModel {
	itk := newIndividualTestKeys(addr, port)
	tk.Runs = append(tk.Runs, itk)
	return itk
}

// IndividualTestKeys contains the experiment results for a single IP/port combo
type IndividualTestKeys struct {
	TCPConnect   []*model.ArchivalTCPConnectResult       `json:"tcp_connect"`
	TLSHandshake *model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Failure      string                                  `json:"failure"`
	FailureStep  string                                  `json:"failed_step"`
	IP           string                                  `json:"ip"`
	Port         string                                  `json:"port"`
	noopCounter  uint8
}

func newIndividualTestKeys(addr string, port string) *IndividualTestKeys {
	itk := new(IndividualTestKeys)
	itk.IP = addr
	itk.Port = port
	return itk
}

// IPPort TCPSessionModel
func (itk *IndividualTestKeys) IPPort(ip string, port string) {
	itk.IP = ip
	itk.Port = port
}

// ConnectResults TCPSessionModel
func (itk *IndividualTestKeys) ConnectResults(res []*model.ArchivalTCPConnectResult) {
	itk.TCPConnect = append(itk.TCPConnect, res...)
}

// HandshakeResult TCPSessionModel
func (itk *IndividualTestKeys) HandshakeResult(res *model.ArchivalTLSOrQUICHandshakeResult) {
	itk.TLSHandshake = res
}

// FailedStep TCPSessionModel
func (itk *IndividualTestKeys) FailedStep(failure string, step string) {
	itk.Failure = failure
	itk.FailureStep = step
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

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	sess := args.Session
	measurement := args.Measurement
	log := sess.Logger()
	trace := measurexlite.NewTrace(0, measurement.MeasurementStartTimeSaved)

	config, err := config(measurement.Input)
	if err != nil {
		// Invalid input data, we don't even generate report
		return err
	}

	tk := newTestKeys(config.host)
	measurement.TestKeys = tk

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tlsconfig := tls.Config{
		InsecureSkipVerify: false,
		ServerName:         config.host,
	}

	runner := &tcprunner.TCPRunner{
		Tk:        tk,
		Trace:     trace,
		Logger:    log,
		Ctx:       ctx,
		Tlsconfig: &tlsconfig,
	}

	// First resolve DNS
	addrs, success := runner.Resolve(config.host)
	if !success {
		return nil
	}

	for _, addr := range addrs {
		tcpSession, success := runner.Conn(addr, config.port)
		if !success {
			continue
		}
		defer tcpSession.Close()

		if config.forcedTLS {
			log.Infof("Running direct TLS mode to %s:%s", addr, config.port)

			if !tcpSession.Handshake() {
				continue
			}

			// Try NoOps
			if !testIMAP(tcpSession, config.noopCount) {
				continue
			}
		} else {
			log.Infof("Running StartTLS mode to %s:%s", addr, config.port)

			// Upgrade via StartTLS and try NoOps
			if !tcpSession.StartTLS("A1 STARTTLS\n", "TLS") {
				continue
			}

			if !testIMAP(tcpSession, config.noopCount) {
				continue
			}
		}
	}

	return nil
}

func testIMAP(s *tcprunner.TCPSession, noop uint8) bool {
	// Auto-choose plaintext/TCP session
	// TODO: move to Debugf
	s.Runner.Logger.Infof("Retrieving existing connection")
	conn := s.CurrentConn()
	s.Runner.Logger.Infof("Starting IMAP query")

	command, err := bufio.NewReader(conn).ReadString('\n')
	if err != nil {
		s.FailedStep(*tracex.NewFailure(err), "imap_wait_capability")
		return false
	}

	if !strings.Contains(command, "CAPABILITY") {
		s.FailedStep(fmt.Sprintf("Received unexpected IMAP response: %s", command), "imap_wrong_capability")
		return false
	}

	s.Runner.Logger.Infof("Finished starting IMAP")

	if noop > 0 {
		// Downcast TCPSession's itk into typed IndividualTestKeys to access noopCounter field
		concreteITK := s.Itk.(*IndividualTestKeys)
		s.Runner.Logger.Infof("Trying to generate more no-op traffic")
		concreteITK.noopCounter = 0
		for concreteITK.noopCounter < noop {
			concreteITK.noopCounter++
			s.Runner.Logger.Infof("NoOp Iteration %d", concreteITK.noopCounter)
			_, err = conn.Write([]byte("A1 NOOP\n"))
			if err != nil {
				s.FailedStep(*tracex.NewFailure(err), fmt.Sprintf("imap_noop_%d", concreteITK.noopCounter))
				break
			}
		}

		if concreteITK.noopCounter == noop {
			s.Runner.Logger.Infof("Successfully generated no-op traffic")
			return true
		}
		s.Runner.Logger.Warnf("Failed no-op traffic at iteration %d", concreteITK.noopCounter)
		return false
	}

	return true
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
	//DNSBlocking bool `json:"facebook_dns_blocking"`
	//TCPBlocking bool `json:"facebook_tcp_blocking"`
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
