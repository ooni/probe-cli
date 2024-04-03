package openvpn

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"strings"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
)

var (
	ErrBadBase64Blob = errors.New("wrong base64 encoding")
)

// endpoint is a single endpoint to be probed.
// The information contained in here is not sufficient to complete a connection:
// we need to augment it with more info, as cipher selection or obfuscating proxy credentials.
type endpoint struct {
	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// Obfuscation is any obfuscation method use to connect to this endpoint.
	// Valid values are: obfs4, none.
	Obfuscation string

	// Port is the Port for this endpoint.
	Port string

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

// String implements Stringer. This is a compact representation of the endpoint,
// which differs from the input URI scheme. This is the canonical representation, that can be used
// to deterministically slice a list of endpoints, sort them lexicographically, etc.
func (e *endpoint) String() string {
	var proto string
	if e.Obfuscation == "obfs4" {
		proto = e.Protocol + "+obfs4"
	} else {
		proto = e.Protocol
	}
	return fmt.Sprintf("%s://%s:%s/%s", proto, e.IPAddr, e.Port, e.Transport)
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

	return fmt.Sprintf(
		"%s://%s.corp/?address=%s:%s&transport=%s",
		proto, provider, e.IPAddr, e.Port, e.Transport)
}

// endpointList is a list of endpoints.
type endpointList []*endpoint

// DefaultEndpoints contains a subset of known endpoints to be used if no input is passed to the experiment and
// the backend query fails for whatever reason. We risk distributing endpoints that can go stale, so we should be careful about
// the stability of the endpoints selected here, but in restrictive environments it's useful to have something
// to probe in absence of an useful OONI API. Valid credentials are still needed, though.
var DefaultEndpoints = endpointList{
	{
		Provider:  "riseup",
		IPAddr:    "51.15.187.53",
		Port:      "1194",
		Protocol:  "openvpn",
		Transport: "tcp",
	},
	{
		Provider:  "riseup",
		IPAddr:    "51.15.187.53",
		Port:      "1194",
		Protocol:  "openvpn",
		Transport: "udp",
	},
}

// Shuffle returns a shuffled copy of the endpointList.
func (e endpointList) Shuffle() endpointList {
	rand.Shuffle(len(e), func(i, j int) {
		e[i], e[j] = e[j], e[i]
	})
	return e
}

// defaultOptionsByProvider is a map containing base config for
// all the known providers. We extend this base config with credentials coming
// from the OONI API.
var defaultOptionsByProvider = map[string]*vpnconfig.OpenVPNOptions{
	"riseup": {
		Auth:   "SHA512",
		Cipher: "AES-256-GCM",
	},
}

// APIEnabledProviders is the list of providers that the stable API Endpoint knows about.
// This array will be a subset of the keys in defaultOptionsByProvider, but it might make sense
// to still register info about more providers that the API officially knows about.
var APIEnabledProviders = []string{
	"riseup",
}

// isValidProvider returns true if the provider is found as key in the registry of defaultOptionsByProvider.
// TODO(ainghazal): consolidate with list of enabled providers from the API viewpoint.
func isValidProvider(provider string) bool {
	_, ok := defaultOptionsByProvider[provider]
	return ok
}

// getOpenVPNConfig gets a properly configured [*vpnconfig.Config] object for the given endpoint.
// To obtain that, we merge the endpoint specific configuration with base options.
// Base options are hardcoded for the moment, for comparability among different providers.
// We can add them to the OONI API and as extra cli options if ever needed.
func getOpenVPNConfig(
	tracer *vpntracex.Tracer,
	logger model.Logger,
	endpoint *endpoint,
	creds *vpnconfig.OpenVPNOptions) (*vpnconfig.Config, error) {
	// TODO(ainghazal): use merge ability in vpnconfig.OpenVPNOptions merge (pending PR)
	provider := endpoint.Provider
	if !isValidProvider(provider) {
		return nil, fmt.Errorf("%w: unknown provider: %s", ErrInvalidInput, provider)
	}
	baseOptions := defaultOptionsByProvider[provider]

	cfg := vpnconfig.NewConfig(
		vpnconfig.WithLogger(logger),
		vpnconfig.WithOpenVPNOptions(
			&vpnconfig.OpenVPNOptions{
				// endpoint-specific options.
				Remote: endpoint.IPAddr,
				Port:   endpoint.Port,
				Proto:  vpnconfig.Proto(endpoint.Transport),

				// options coming from the default known values.
				Cipher: baseOptions.Cipher,
				Auth:   baseOptions.Auth,

				// auth coming from passed credentials.
				CA:   creds.CA,
				Cert: creds.Cert,
				Key:  creds.Key,
			},
		),
		vpnconfig.WithHandshakeTracer(tracer),
	)

	return cfg, nil
}

// extractBase64Blob is used to pass credentials as command-line options.
func extractBase64Blob(val string) (string, error) {
	s := strings.TrimPrefix(val, "base64:")
	if len(s) == len(val) {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, "missing prefix")
	}
	dec, err := base64.URLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, err)
	}
	return string(dec), nil
}

func isValidProtocol(s string) bool {
	if strings.HasPrefix(s, "openvpn://") {
		return true
	}
	if strings.HasPrefix(s, "openvpn+obfs4://") {
		return true
	}
	return false
}
