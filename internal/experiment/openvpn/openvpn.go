// Package openvpn contains a generic openvpn experiment.
package openvpn

import (
	"context"
	"errors"
	"fmt"
	"strconv"
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

var (
	ErrBadAuth = errors.New("bad provider authentication")
)

// Config contains the experiment config.
//
// This contains all the settings that user can set to modify the behaviour
// of this experiment. By tagging these variables with `ooni:"..."`, we allow
// miniooni's -O flag to find them and set them.
type Config struct {
	// TODO(ainghazal): Provider is right now ignored. InputLoader should get the provider from options.
	Provider string `ooni:"VPN provider"`
	SafeKey  string `ooni:"key to connect to the OpenVPN endpoint"`
	SafeCert string `ooni:"cert to connect to the OpenVPN endpoint"`
	SafeCA   string `ooni:"ca to connect to the OpenVPN endpoint"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	Success          bool                              `json:"success"`
	NetworkEvents    []*vpntracex.Event                `json:"network_events"`
	TCPConnect       []*model.ArchivalTCPConnectResult `json:"tcp_connect,omitempty"`
	OpenVPNHandshake []*ArchivalOpenVPNHandshakeResult `json:"openvpn_handshake"`
}

// NewTestKeys creates new openvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		Success:          false,
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

// AllConnectionsSuccessful returns true if all the registered handshakes have Status.Success equal to true.
func (tk *TestKeys) AllConnectionsSuccessful() bool {
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
	ErrInvalidInput = errors.New("invalid input")
)

func parseEndpoint(m *model.Measurement) (*endpoint, error) {
	if m.Input != "" {
		if ok := isValidProtocol(string(m.Input)); !ok {
			return nil, ErrInvalidInput
		}
		return newEndpointFromInputString(string(m.Input))
	}
	// The current InputPolicy should ensure we have a hardcoded input,
	// so this error should only be raised if by mistake we change the InputPolicy.
	return nil, fmt.Errorf("%w: %s", ErrInvalidInput, "input is mandatory")
}

// AuthMethod is the authentication method used by a provider.
type AuthMethod string

var (
	AuthCertificate = AuthMethod("cert")
	AuthUserPass    = AuthMethod("userpass")
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

	method, ok := providerAuthentication[provider]
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
	sess.Logger().Infof("Probing endpoint %s", endpoint.String())

	// TODO:  separate pre-connection checks
	connResult, err := m.connectAndHandshake(ctx, int64(1), time.Now(), sess, endpoint)
	if err != nil {
		sess.Logger().Warn("Fatal error while attempting to connect to endpoint, aborting!")
		return err
	}
	if connResult != nil {
		tk.AddConnectionTestKeys(connResult)
	}
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

// connectAndHandshake dials a connection and attempts an OpenVPN handshake using that dialer.
func (m *Measurer) connectAndHandshake(ctx context.Context, index int64, zeroTime time.Time, sess model.ExperimentSession, endpoint *endpoint) (*SingleConnection, error) {

	logger := sess.Logger()

	// create a trace for the network dialer
	trace := measurexlite.NewTrace(index, zeroTime)

	dialer := trace.NewDialerWithoutResolver(logger)

	// create a vpn tun Device that attempts to dial and performs the handshake
	handshakeTracer := vpntracex.NewTracerWithTransactionID(zeroTime, index)

	// TODO -- move to outer function ------
	credentials, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, endpoint.Provider)
	if err != nil {
		return nil, err
	}

	openvpnConfig, err := getOpenVPNConfig(handshakeTracer, endpoint, credentials)
	if err != nil {
		return nil, err
	}
	// TODO -- move to outer function ------

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
		bootstrapTime = tLast - tFirst
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
			T0:            tFirst,
			T:             tLast,
			Tags:          []string{},
			TransactionID: index,
		},
		NetworkEvents: handshakeEvents,
	}, nil
}

// TODO: get cached from session instead of fetching every time
func (m *Measurer) FetchProviderCredentials(
	ctx context.Context,
	sess model.ExperimentSession,
	provider string) (*model.OOAPIVPNProviderConfig, error) {
	// TODO(ainghazal): do pass country code, can be useful to orchestrate campaigns specific to areas
	config, err := sess.FetchOpenVPNConfig(ctx, provider, "XX")
	if err != nil {
		return &model.OOAPIVPNProviderConfig{}, err
	}
	return config, nil
}
