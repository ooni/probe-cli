package openvpn

import (
	"fmt"
	"net"
	"net/url"
	"slices"
	"strings"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

var (
	ErrInputRequired = targetloading.ErrInputRequired
	ErrInvalidInput  = targetloading.ErrInvalidInput
)

// endpoint is a single endpoint to be probed.
// The information contained in here is not sufficient to complete a connection:
// we need to augment it with more info, as cipher selection or obfuscating proxy credentials.
type endpoint struct {
	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// DomainName is an optional domain name that we use internally to get the IP address.
	// This is just a convenience field, the experiments should always be done against a canonical IPAddr.
	DomainName string

	// Obfuscation is any obfuscation method use to connect to this endpoint.
	// Valid values are: obfs4, none.
	Obfuscation string

	// Port is the Port for this endpoint.
	Port string

	// PreferredCountries is an optional array of country codes. Probes in these countries have preference on this
	// endpoint.
	PreferredCountries []string

	// Protocol is the tunneling protocol (openvpn, openvpn+obfs4).
	Protocol string

	// Provider is a unique label identifying the provider maintaining this endpoint.
	Provider string

	// Transport is the underlying transport used for this endpoint. Valid transports are `tcp` and `udp`.
	Transport string
}

// newEndpointFromInputString constructs an endpoint after parsing an input string.
//
// The input URI is in the form:
// "openvpn://provider.corp/?address=1.2.3.4:1194&transport=udp
// "openvpn+obfs4://provider.corp/address=1.2.3.4:1194?&cert=deadbeef&iat=0"
func newEndpointFromInputString(uri string) (*endpoint, error) {
	if uri == "" {
		return nil, ErrInputRequired
	}
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	var obfuscation string
	switch parsedURL.Scheme {
	case "openvpn":
		obfuscation = "none"
	case "openvpn+obfs4":
		obfuscation = "obfs4"
	default:
		return nil, fmt.Errorf("%w: unknown scheme: %s", ErrInvalidInput, parsedURL.Scheme)
	}

	provider := strings.TrimSuffix(parsedURL.Hostname(), ".corp")
	if provider == "" {
		return nil, fmt.Errorf("%w: expected provider as host: %s", ErrInvalidInput, parsedURL.Host)
	}
	if !isValidProvider(provider) {
		return nil, fmt.Errorf("%w: unknown provider: %s", ErrInvalidInput, provider)
	}

	params := parsedURL.Query()

	transport := params.Get("transport")
	if transport != "tcp" && transport != "udp" {
		return nil, fmt.Errorf("%w: invalid transport: %s", ErrInvalidInput, transport)
	}

	address := params.Get("address")
	if address == "" {
		return nil, fmt.Errorf("%w: please specify an address as part of the input", ErrInvalidInput)
	}
	ip, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("%w: cannot split ip:port", ErrInvalidInput)
	}
	if parsedIP := net.ParseIP(ip); parsedIP == nil {
		return nil, fmt.Errorf("%w: bad ip", ErrInvalidInput)
	}

	endpoint := &endpoint{
		IPAddr:      ip,
		Port:        port,
		Obfuscation: obfuscation,
		Protocol:    "openvpn",
		Provider:    provider,
		Transport:   transport,
	}
	return endpoint, nil
}

// String implements [fmt.Stringer]. This is a compact representation of the endpoint,
// which differs from the input URI scheme. This is the canonical representation, that can be used
// to deterministically slice a list of endpoints, sort them lexicographically, etc.
func (e *endpoint) String() string {
	var proto string
	if e.Obfuscation == "obfs4" {
		proto = e.Protocol + "+obfs4"
	} else {
		proto = e.Protocol
	}
	url := &url.URL{
		Scheme: proto,
		Host:   net.JoinHostPort(e.IPAddr, e.Port),
		Path:   e.Transport,
	}
	return url.String()
}

// AsInputURI is a string representation of this endpoint, as used in the experiment input URI format.
func (e *endpoint) AsInputURI() string {
	var proto string
	if e.Obfuscation == "obfs4" {
		proto = e.Protocol + "+obfs4"
	} else {
		proto = e.Protocol
	}

	provider := e.Provider
	if provider == "" {
		provider = "unknown"
	}

	values := map[string][]string{
		"address":   {net.JoinHostPort(e.IPAddr, e.Port)},
		"transport": {e.Transport},
	}

	url := &url.URL{
		Scheme:   proto,
		Host:     provider + ".corp",
		RawQuery: url.Values(values).Encode(),
	}
	return url.String()
}

// APIEnabledProviders is the list of providers that the stable API Endpoint and/or this
// experiment knows about.
var APIEnabledProviders = []string{
	"riseupvpn",
	"oonivpn",
}

// isValidProvider returns true if the provider is found as key in the array of [APIEnabledProviders].
func isValidProvider(provider string) bool {
	return slices.Contains(APIEnabledProviders, provider)
}

// newOpenVPNConfig returns a properly configured [*vpnconfig.Config] object for the given endpoint.
// To obtain that, we merge the endpoint specific configuration with the options passed as richer input targets.
func newOpenVPNConfig(
	tracer *vpntracex.Tracer,
	logger model.Logger,
	endpoint *endpoint,
	config *Config) (*vpnconfig.Config, error) {

	provider := endpoint.Provider
	if !isValidProvider(provider) {
		return nil, fmt.Errorf("%w: unknown provider: %s", ErrInvalidInput, provider)
	}

	cfg := vpnconfig.NewConfig(
		vpnconfig.WithLogger(logger),
		vpnconfig.WithOpenVPNOptions(
			&vpnconfig.OpenVPNOptions{
				// endpoint-specific options.
				Remote:   endpoint.IPAddr,
				Port:     endpoint.Port,
				Proto:    vpnconfig.Proto(endpoint.Transport),
				Compress: vpnconfig.Compression(config.Compress),

				// options and credentials come from the experiment
				// richer input targets.
				Cipher: config.Cipher,
				Auth:   config.Auth,
				CA:     []byte(config.SafeCA),
				Cert:   []byte(config.SafeCert),
				Key:    []byte(config.SafeKey),
			},
		),
		vpnconfig.WithHandshakeTracer(tracer),
	)

	return cfg, nil
}
