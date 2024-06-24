// Package openvpn contains a generic openvpn experiment.
package openvpn

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/minivpn/pkg/tunnel"
)

const (
	testName        = "openvpn"
	testVersion     = "0.1.3"
	openVPNProtocol = "openvpn"
)

// errors are in addition to any other errors returned by the low level packages
// that are used by this experiment to implement its functionality.
var (
	// ErrInputRequired is returned when the experiment is not passed any input.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInput is returned if we failed to parse the input to obtain an endpoint we can measure.
	ErrInvalidInput = errors.New("invalid input")
)

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
// TODO(ainghazal): do pass Auth, Cipher and Compress to OpenVPN config options.
type Config struct {
	Auth        string `ooni:"OpenVPN authentication to use"`
	Cipher      string `ooni:"OpenVPN cipher to use"`
	Compress    string `ooni:"OpenVPN compression to use"`
	Provider    string `ooni:"VPN provider"`
	Obfuscation string `ooni:"Obfuscation to use (obfs4, none)"`
	SafeKey     string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert    string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCA      string `ooni:"ca to connect to the OpenVPN endpoint"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	Success          bool                                    `json:"success"`
	NetworkEvents    []*vpntracex.Event                      `json:"network_events"`
	TCPConnect       []*model.ArchivalTCPConnectResult       `json:"tcp_connect,omitempty"`
	OpenVPNHandshake []*model.ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
}

// NewTestKeys creates new openvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		Success:          false,
		NetworkEvents:    []*vpntracex.Event{},
		TCPConnect:       []*model.ArchivalTCPConnectResult{},
		OpenVPNHandshake: []*model.ArchivalOpenVPNHandshakeResult{},
	}
}

// SingleConnection contains the results of a single handshake.
type SingleConnection struct {
	TCPConnect       *model.ArchivalTCPConnectResult       `json:"tcp_connect,omitempty"`
	OpenVPNHandshake *model.ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
	NetworkEvents    []*vpntracex.Event                    `json:"network_events"`
	// TODO(ainghazal): make sure to document in the spec that these network events only cover the handshake.
	// TODO(ainghazal): in the future, we will want to store more operations under this struct for a single connection,
	// like pingResults or urlgetter calls.
}

// AddConnectionTestKeys adds the result of a single OpenVPN connection attempt to the
// corresponding array in the [TestKeys] object.
func (tk *TestKeys) AddConnectionTestKeys(result *SingleConnection) {
	// Note that TCPConnect is nil when we're using UDP.
	if result.TCPConnect != nil {
		tk.TCPConnect = append(tk.TCPConnect, result.TCPConnect)
	}
	tk.OpenVPNHandshake = append(tk.OpenVPNHandshake, result.OpenVPNHandshake)
	tk.NetworkEvents = append(tk.NetworkEvents, result.NetworkEvents...)
}

// AllConnectionsSuccessful returns true if all the registered handshakes have nil failures.
func (tk *TestKeys) AllConnectionsSuccessful() bool {
	if len(tk.OpenVPNHandshake) == 0 {
		return false
	}
	for _, c := range tk.OpenVPNHandshake {
		if c.Failure != nil {
			return false
		}
	}
	return true
}

// Measurer performs the measurement.
type Measurer struct {
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return Measurer{}
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// AuthMethod is the authentication method used by a provider.
type AuthMethod string

var (
	// AuthCertificate is used for providers that authenticate clients via certificates.
	AuthCertificate = AuthMethod("cert")

	// AuthUserPass is used for providers that authenticate clients via username (or token) and password.
	AuthUserPass = AuthMethod("userpass")
)

// connectAndHandshake dials a connection and attempts an OpenVPN handshake using that dialer.
func (m *Measurer) connectAndHandshake(
	ctx context.Context,
	zeroTime time.Time,
	index int64,
	logger model.Logger,
	endpoint *endpoint,
	openvpnConfig *vpnconfig.Config,
	handshakeTracer *vpntracex.Tracer) *SingleConnection {

	// create a trace for the network dialer
	trace := measurexlite.NewTrace(index, zeroTime)
	dialer := trace.NewDialerWithoutResolver(logger)

	// Create a vpn tun Device that attempts to dial and performs the handshake.
	// Any error will be returned as a failure in the SingleConnection result.
	tun, err := tunnel.Start(ctx, dialer, openvpnConfig)
	if tun != nil {
		defer tun.Close()
	}

	handshakeEvents := handshakeTracer.Trace()
	port, _ := strconv.Atoi(endpoint.Port)

	var (
		tFirst        float64
		tLast         float64
		bootstrapTime float64
	)

	if len(handshakeEvents) > 0 {
		tFirst = handshakeEvents[0].AtTime
		tLast = handshakeEvents[len(handshakeEvents)-1].AtTime
		bootstrapTime = tLast - tFirst
	}

	return &SingleConnection{
		TCPConnect: trace.FirstTCPConnectOrNil(),
		OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
			BootstrapTime: bootstrapTime,
			Endpoint:      endpoint.String(),
			Failure:       measurexlite.NewFailure(err),
			IP:            endpoint.IPAddr,
			Port:          port,
			Transport:     endpoint.Transport,
			Provider:      endpoint.Provider,
			OpenVPNOptions: model.ArchivalOpenVPNOptions{
				Cipher:      openvpnConfig.OpenVPNOptions().Cipher,
				Auth:        openvpnConfig.OpenVPNOptions().Auth,
				Compression: string(openvpnConfig.OpenVPNOptions().Compress),
			},
			T0:            tFirst,
			T:             tLast,
			Tags:          []string{},
			TransactionID: index,
		},
		NetworkEvents: handshakeEvents,
	}
}

// FetchProviderCredentials will extract credentials from the configuration we gathered for a given provider.
func (m *Measurer) FetchProviderCredentials(
	ctx context.Context,
	sess model.ExperimentSession,
	provider string) (*model.OOAPIVPNProviderConfig, error) {
	// TODO(ainghazal): pass real country code, can be useful to orchestrate campaigns specific to areas.
	// Since we have contacted the API previously, this call should use the cached info contained in the session.
	config, err := sess.FetchOpenVPNConfig(ctx, provider, "XX")
	if err != nil {
		return nil, err
	}
	return config, nil
}

// Run implements model.ExperimentMeasurer.Run.
// A single run expects exactly ONE input (endpoint), but we can modify whether
// to test different transports by settings options.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// 0. obtain the richer input target, config, and input or panic
	if args.Target == nil {
		return ErrInputRequired
	}

	tk := NewTestKeys()

	zeroTime := time.Now()
	idx := int64(1)
	handshakeTracer := vpntracex.NewTracerWithTransactionID(zeroTime, idx)

	// build the input
	target := args.Target.(*Target)
	config, input := target.Options, target.URL
	sess.Logger().Infof("openvpn: using richer input: %+v", input)

	endpoint, err := newEndpointFromInputString(input)
	if err != nil {
		return err
	}

	openvpnConfig, err := mergeOpenVPNConfig(handshakeTracer, sess.Logger(), endpoint, config)
	if err != nil {
		return err
	}

	sess.Logger().Infof("Probing endpoint %s", endpoint.String())

	connResult := m.connectAndHandshake(ctx, zeroTime, idx, sess.Logger(), endpoint, openvpnConfig, handshakeTracer)
	tk.AddConnectionTestKeys(connResult)
	tk.Success = tk.AllConnectionsSuccessful()

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
