// Package openvpn contains a generic openvpn experiment.
package openvpn

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"

	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/minivpn/pkg/tunnel"
)

const (
	testVersion   = "0.1.0"
	openVPNProcol = "openvpn"
)

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
type Config struct {
	Provider string `ooni:"VPN provider"`
	SafeKey  string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCA   string `ooni:"ca to connect to the OpenVPN endpoint"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	AllSuccess       bool                              `json:"success_all"`
	AnySuccess       bool                              `json:"success_any"`
	Inputs           []string                          `json:"inputs"`
	NetworkEvents    []*vpntracex.Event                `json:"network_events"`
	TCPConnect       []*model.ArchivalTCPConnectResult `json:"tcp_connect,omitempty"`
	OpenVPNHandshake []*ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
}

// NewTestKeys creates new openvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		AllSuccess:       false,
		AnySuccess:       false,
		Inputs:           []string{},
		NetworkEvents:    []*vpntracex.Event{},
		TCPConnect:       []*model.ArchivalTCPConnectResult{},
		OpenVPNHandshake: []*ArchivalOpenVPNHandshakeResult{},
	}
}

// SingleConnection contains the results of a single handshake.
type SingleConnection struct {
	TCPConnect       *model.ArchivalTCPConnectResult `json:"tcp_connect,omitempty"`
	OpenVPNHandshake *ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
	NetworkEvents    []*vpntracex.Event              `json:"network_events"`
	// TODO(ainghazal): make sure to document in the spec that these network events only cover the handshake.
	// TODO(ainghazal): in the future, we will want to store more operations under this struct for a single connection,
	// like pingResults or urlgetter calls.
}

// AddConnectionTestKeys adds the result of a single OpenVPN connection attempt to the
// corresponding array in the [TestKeys] object.
func (tk *TestKeys) AddConnectionTestKeys(result *SingleConnection) {
	if result.TCPConnect != nil {
		tk.TCPConnect = append(tk.TCPConnect, result.TCPConnect)
	}
	tk.OpenVPNHandshake = append(tk.OpenVPNHandshake, result.OpenVPNHandshake)
	tk.NetworkEvents = append(tk.NetworkEvents, result.NetworkEvents...)
}

// allConnectionsSuccessful returns true if all the registered handshakes have Status.Success equal to true.
func (tk *TestKeys) allConnectionsSuccessful() bool {
	for _, c := range tk.OpenVPNHandshake {
		if !c.Status.Success {
			return false
		}
	}
	return true
}

// anyConnectionSuccessful returns true if any of the registered handshakes have Status.Success equal to true.
func (tk *TestKeys) anyConnectionSuccessful() bool {
	for _, c := range tk.OpenVPNHandshake {
		if c.Status.Success {
			return true
		}
	}
	return false
}

// Measurer performs the measurement.
type Measurer struct {
	config   Config
	testName string
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config, testName string) model.ExperimentMeasurer {
	// TODO(ainghazal): allow ooniprobe to override this.
	config.Provider = "riseup"
	return Measurer{config: config, testName: testName}
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return m.testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

var (
	ErrInvalidInput = errors.New("invalid input")
)

func parseListOfInputs(inputs string) (endpointList, error) {
	endpoints := make(endpointList, 0)
	inputList := strings.Split(inputs, ",")
	for _, i := range inputList {
		e, err := newEndpointFromInputString(i)
		if err != nil {
			return endpoints, err
		}
		endpoints = append(endpoints, e)
	}
	return endpoints, nil

}

// ErrFailure is the error returned when you set the
// config.ReturnError field to true.
var ErrFailure = errors.New("mocked error")

// Run implements model.ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	var endpoints endpointList
	var err error

	if measurement.Input == "" {
		// if input is null, we get the hardcoded list of inputs.
		endpoints = allEndpoints
	} else {
		// otherwise, we expect a comma-separated value of inputs in
		// the URI scheme defined for openvpn experiments.
		endpoints, err = parseListOfInputs(string(measurement.Input))
		if err != nil {
			return err
		}
	}

	tk := NewTestKeys()

	// TODO(ainghazal): we could parallelize multiple probing.
	for idx, endpoint := range endpoints.Shuffle() {
		sess.Logger().Infof("Probing endpoint %s", endpoint.String())
		tk.Inputs = append(tk.Inputs, endpoint.AsInputURI())

		connResult := m.connectAndHandshake(ctx, int64(idx+1), time.Now(), sess.Logger(), endpoint)
		// TODO: better catch error here.
		if connResult != nil {
			tk.AddConnectionTestKeys(connResult)
		}
	}
	tk.AllSuccess = tk.allConnectionsSuccessful()
	tk.AnySuccess = tk.anyConnectionSuccessful()

	callbacks.OnProgress(1.0, "All endpoints probed")
	measurement.TestKeys = tk

	// TODO(ainghazal): validate we have valid config for each endpoint.
	// TODO(ainghazal): validate hostname is a valid IP (ipv4 or 6)
	// TODO(ainghazal): decide what to do if we have expired certs (abort one measurement or abort the whole experiment?)

	// Note: if here we return an error, the parent code will assume
	// something fundamental was wrong and we don't have a measurement
	// to submit to the OONI collector. Keep this in mind when you
	// are writing new experiments!
	return nil
}

// connectAndHandshake dials a connection and attempts an OpenVPN handshake using that dialer.
func (m *Measurer) connectAndHandshake(ctx context.Context, index int64,
	zeroTime time.Time, logger model.Logger, endpoint *endpoint) *SingleConnection {

	// create a trace for the network dialer
	trace := measurexlite.NewTrace(index, zeroTime)

	// TODO(ainghazal): can I pass tags to this tracer?
	dialer := trace.NewDialerWithoutResolver(logger)

	// create a vpn tun Device that attempts to dial and performs the handshake
	handshakeTracer := vpntracex.NewTracerWithTransactionID(zeroTime, index)

	openvpnConfig, err := getVPNConfig(handshakeTracer, endpoint, &m.config)
	if err != nil {
		// TODO: find a better way to return the error - this is not a test failure,
		// it's a failure to start the measurement. we should abort
		return nil
	}
	tun, err := tunnel.Start(ctx, dialer, openvpnConfig)

	var failure string
	if err != nil {
		failure = err.Error()
	}
	defer tun.Close()

	handshakeEvents := handshakeTracer.Trace()
	port, _ := strconv.Atoi(endpoint.Port)

	var (
		tFirst        float64
		tLast         float64
		bootstrapTime float64
	)

	if len(handshakeEvents) != 0 {
		tFirst = handshakeEvents[0].AtTime
		tLast = handshakeEvents[len(handshakeEvents)-1].AtTime
		bootstrapTime = time.Since(zeroTime).Seconds()
	}

	return &SingleConnection{
		TCPConnect: trace.FirstTCPConnectOrNil(),
		OpenVPNHandshake: &ArchivalOpenVPNHandshakeResult{
			BootstrapTime: bootstrapTime,
			Endpoint:      endpoint.String(),
			IP:            endpoint.IPAddr,
			Port:          port,
			Transport:     endpoint.Transport,
			Provider:      endpoint.Provider,
			OpenVPNOptions: OpenVPNOptions{
				Cipher:      openvpnConfig.OpenVPNOptions().Cipher,
				Auth:        openvpnConfig.OpenVPNOptions().Auth,
				Compression: string(openvpnConfig.OpenVPNOptions().Compress),
			},
			Status: ArchivalOpenVPNConnectStatus{
				Failure: &failure,
				Success: err == nil,
			},
			StartTime:     zeroTime,
			T0:            tFirst,
			T:             tLast,
			Tags:          []string{},
			TransactionID: index,
		},
		NetworkEvents: handshakeEvents,
	}
}
