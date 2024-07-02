// Package openvpn contains a generic openvpn experiment.
package openvpn

import (
	"context"
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
	testVersion     = "0.1.4"
	openVPNProtocol = "openvpn"
)

var (
	ErrInvalidInputType = targetloading.ErrInvalidInputType
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
	BootstrapTime    float64                                 `json:"bootstrap_time,omitempty"`
	Tunnel           string                                  `json:"tunnel"`
	Failure          *string                                 `json:"failure"`
}

// NewTestKeys creates new openvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		Success:          false,
		NetworkEvents:    []*vpntracex.Event{},
		TCPConnect:       []*model.ArchivalTCPConnectResult{},
		OpenVPNHandshake: []*model.ArchivalOpenVPNHandshakeResult{},
		BootstrapTime:    0,
		Tunnel:           "openvpn",
		Failure:          nil,
	}
}

// SingleConnection contains the results of a single handshake.
type SingleConnection struct {
	BootstrapTime    float64
	TCPConnect       *model.ArchivalTCPConnectResult       `json:"tcp_connect,omitempty"`
	OpenVPNHandshake *model.ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
	NetworkEvents    []*vpntracex.Event                    `json:"network_events"`
	// TODO(ainghazal): in the future, we will want to store more operations under this struct for a single connection,
	// like pingResults or urlgetter calls. Be sure to modify the spec when that happens.
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

	// we assume one measurement has exactly one effective connection
	tk.BootstrapTime = result.BootstrapTime

	if result.OpenVPNHandshake.Failure != nil {
		tk.Failure = result.OpenVPNHandshake.Failure
		tk.BootstrapTime = 0
	}
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
type Measurer struct{}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{}
}

// ExperimentName implements model.ExperimentMeasurer.ExperimentName.
func (m *Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements model.ExperimentMeasurer.ExperimentVersion.
func (m *Measurer) ExperimentVersion() string {
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

	t0, t, handshakeTime := TimestampsFromHandshake(handshakeEvents)

	// the bootstrap time is defined to be zero if there's a handshake failure.
	var bootstrapTime float64
	if err == nil {
		bootstrapTime = time.Since(zeroTime).Seconds()
	}

	return &SingleConnection{
		BootstrapTime: bootstrapTime,
		TCPConnect:    trace.FirstTCPConnectOrNil(),
		OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
			HandshakeTime: handshakeTime,
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
			T0:            t0,
			T:             t,
			Tags:          []string{},
			TransactionID: index,
		},
		NetworkEvents: handshakeEvents,
	}
}

// TimestampsFromHandshake returns the t0, t and duration of the handshake events.
// If the passed events are a zero-len array, all of the results will be zero.
func TimestampsFromHandshake(events []*vpntracex.Event) (float64, float64, float64) {
	var (
		t0       float64
		t        float64
		duration float64
	)
	if len(events) > 0 {
		t0 = events[0].AtTime
		t = events[len(events)-1].AtTime
		duration = t - t0
	}
	return t0, t, duration
}

// TODO: delete-me
// FetchProviderCredentials will extract credentials from the configuration we gathered for a given provider.
/*
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
*/

// Run implements model.ExperimentMeasurer.Run.
// A single run expects exactly ONE input (endpoint), but we can modify whether
// to test different transports by settings options.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	// 0. fail if there's no richer input target
	if args.Target == nil {
		return targetloading.ErrInputRequired
	}

	tk := NewTestKeys()

	zeroTime := time.Now()
	idx := int64(1)
	handshakeTracer := vpntracex.NewTracerWithTransactionID(zeroTime, idx)

	// 1. build the input
	target, ok := args.Target.(*Target)
	if !ok {
		return targetloading.ErrInvalidInputType
	}
	config, input := target.Config, target.URL

	// 2. obtain the endpoint representation from the input URL
	endpoint, err := newEndpointFromInputString(input)
	if err != nil {
		return err
	}

	// TODO(ainghazal): validate we have valid config for each endpoint.
	// TODO(ainghazal): validate hostname is a valid IP (ipv4 or 6)
	// TODO(ainghazal): decide what to do if we have expired certs (abort one measurement or abort the whole experiment?)

	// 3. build openvpn config from endpoint and options
	openvpnConfig, err := newOpenVPNConfig(handshakeTracer, sess.Logger(), endpoint, config)
	if err != nil {
		return err
	}
	sess.Logger().Infof("Probing endpoint %s", endpoint.String())

	// 4. initiate openvpn handshake against endpoint
	connResult := m.connectAndHandshake(ctx, zeroTime, idx, sess.Logger(), endpoint, openvpnConfig, handshakeTracer)
	tk.AddConnectionTestKeys(connResult)
	tk.Success = tk.AllConnectionsSuccessful()

	callbacks.OnProgress(1.0, "All endpoints probed")

	// 5. assign the testkeys
	measurement.TestKeys = tk

	// Note: if here we return an error, the parent code will assume
	// something fundamental was wrong and we don't have a measurement
	// to submit to the OONI collector. Keep this in mind when you
	// are writing new experiments!
	return nil
}
