// Package openvpn contains a generic openvpn experiment.
package openvpn

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/minivpn/pkg/tunnel"
)

const (
	testVersion   = "0.1.1"
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

// AllConnectionsSuccessful returns true if all the registered handshakes have Status.Success equal to true.
func (tk *TestKeys) AllConnectionsSuccessful() bool {
	if len(tk.OpenVPNHandshake) == 0 {
		return false
	}
	for _, c := range tk.OpenVPNHandshake {
		if !c.Status.Success {
			return false
		}
	}
	return true
}

// Measurer performs the measurement.
type Measurer struct {
	config   Config
	testName string
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config, testName string) model.ExperimentMeasurer {
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
	// ErrInvalidInput is returned if we failed to parse the input to obtain an endpoint we can measure.
	ErrInvalidInput = errors.New("invalid input")
)

func parseEndpoint(m *model.Measurement) (*endpoint, error) {
	if m.Input != "" {
		if ok := isValidProtocol(string(m.Input)); !ok {
			return nil, ErrInvalidInput
		}
		return newEndpointFromInputString(string(m.Input))
	}
	return nil, fmt.Errorf("%w: %s", ErrInvalidInput, "input is mandatory")
}

// AuthMethod is the authentication method used by a provider.
type AuthMethod string

var (
	// AuthCertificate is used for providers that authenticate clients via certificates.
	AuthCertificate = AuthMethod("cert")

	// AuthUserPass is used for providers that authenticate clients via username (or token) and password.
	AuthUserPass = AuthMethod("userpass")
)

var providerAuthentication = map[string]AuthMethod{
	"riseup":     AuthCertificate,
	"tunnelbear": AuthUserPass,
	"surfshark":  AuthUserPass,
}

func hasCredentialsInOptions(cfg Config, method AuthMethod) bool {
	switch method {
	case AuthCertificate:
		ok := cfg.SafeCA != "" && cfg.SafeCert != "" && cfg.SafeKey != ""
		return ok
	default:
		return false
	}
}

// MaybeGetCredentialsFromOptions overrides authentication info with what user provided in options.
// Each certificate/key can be encoded in base64 so that a single option can be safely represented as command line options.
// This function returns no error if there are no credentials in the passed options, only if failing to parse them.
func MaybeGetCredentialsFromOptions(cfg Config, opts *vpnconfig.OpenVPNOptions, method AuthMethod) (bool, error) {
	if ok := hasCredentialsInOptions(cfg, method); !ok {
		return false, nil
	}
	ca, err := extractBase64Blob(cfg.SafeCA)
	if err != nil {
		return false, err
	}
	opts.CA = []byte(ca)

	key, err := extractBase64Blob(cfg.SafeKey)
	if err != nil {
		return false, err
	}
	opts.Key = []byte(key)

	cert, err := extractBase64Blob(cfg.SafeCert)
	if err != nil {
		return false, err
	}
	opts.Cert = []byte(cert)
	return true, nil
}

func (m *Measurer) getCredentialsFromAPI(
	ctx context.Context,
	sess model.ExperimentSession,
	provider string,
	opts *vpnconfig.OpenVPNOptions) error {
	// We expect the credentials from the API response to be encoded as the direct PEM serialization.
	apiCreds, err := m.FetchProviderCredentials(ctx, sess, provider)
	// TODO(ainghazal): validate credentials have the info we expect, certs are not expired etc.
	if err != nil {
		sess.Logger().Warnf("Error fetching credentials from API: %s", err.Error())
		return err
	}
	sess.Logger().Infof("Got credentials from provider: %s", provider)

	opts.CA = []byte(apiCreds.Config.CA)
	opts.Cert = []byte(apiCreds.Config.Cert)
	opts.Key = []byte(apiCreds.Config.Key)
	return nil
}

// GetCredentialsFromOptionsOrAPI attempts to find valid credentials for the given provider, either
// from the passed Options (cli, oonirun), or from a remote call to the OONI API endpoint.
func (m *Measurer) GetCredentialsFromOptionsOrAPI(
	ctx context.Context,
	sess model.ExperimentSession,
	provider string) (*vpnconfig.OpenVPNOptions, error) {

	method, ok := providerAuthentication[strings.TrimSuffix(provider, "vpn")]
	if !ok {
		return nil, fmt.Errorf("%w: provider auth unknown: %s", ErrInvalidInput, provider)
	}

	// Empty options object to fill with credentials.
	creds := &vpnconfig.OpenVPNOptions{}

	switch method {
	case AuthCertificate:
		ok, err := MaybeGetCredentialsFromOptions(m.config, creds, method)
		if err != nil {
			return nil, err
		}
		if ok {
			return creds, nil
		}
		// No options passed, so let's get the credentials that inputbuilder should have cached
		// for us after hitting the OONI API.
		if err := m.getCredentialsFromAPI(ctx, sess, provider, creds); err != nil {
			return nil, err
		}
		return creds, nil

	default:
		return nil, fmt.Errorf("%w: method not implemented (%s)", ErrInvalidInput, method)
	}

}

// mergeOpenVPNConfig attempts to get credentials from Options or API, and then
// constructs a [*vpnconfig.Config] instance after merging the credentials passed by options or API response.
// It also returns an error if the operation fails.
func (m *Measurer) mergeOpenVPNConfig(
	ctx context.Context,
	sess model.ExperimentSession,
	endpoint *endpoint,
	tracer *vpntracex.Tracer) (*vpnconfig.Config, error) {

	logger := sess.Logger()

	credentials, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, endpoint.Provider)
	if err != nil {
		return nil, err
	}

	openvpnConfig, err := getOpenVPNConfig(tracer, logger, endpoint, credentials)
	if err != nil {
		return nil, err
	}
	// TODO(ainghazal): sanity check (Remote, Port, Proto etc + missing certs)
	return openvpnConfig, nil
}

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

	var failure string

	// Create a vpn tun Device that attempts to dial and performs the handshake.
	// Any error will be returned as a failure in the SingleConnection result.
	tun, err := tunnel.Start(ctx, dialer, openvpnConfig)
	if err != nil {
		failure = err.Error()
	}
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
			IP:            endpoint.IPAddr,
			Port:          port,
			Transport:     endpoint.Transport,
			Provider:      endpoint.Provider,
			OpenVPNOptions: model.ArchivalOpenVPNOptions{
				Cipher:      openvpnConfig.OpenVPNOptions().Cipher,
				Auth:        openvpnConfig.OpenVPNOptions().Auth,
				Compression: string(openvpnConfig.OpenVPNOptions().Compress),
			},
			Status: model.ArchivalOpenVPNConnectStatus{
				Failure: &failure,
				Success: err == nil,
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

	endpoint, err := parseEndpoint(measurement)
	if err != nil {
		return err
	}

	tk := NewTestKeys()

	zeroTime := time.Now()
	idx := int64(1)
	handshakeTracer := vpntracex.NewTracerWithTransactionID(zeroTime, idx)

	openvpnConfig, err := m.mergeOpenVPNConfig(ctx, sess, endpoint, handshakeTracer)
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
