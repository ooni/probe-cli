package openvpn

import (
	"encoding/base64"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"strings"

	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

var (
	ErrBadBase64Blob = errors.New("wrong base64 encoding")
)

// endpoint is a single endpoint to be probed.
// The information contained in here is not generally not sufficient to complete a connection:
// we need more info, as cipher selection or obfuscating proxy credentials.
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
// "openvpn://1.2.3.4:443/udp/&provider=tunnelbear"
// "openvpn+obfs4://1.2.3.4:443/tcp/&provider=riseup&cert=deadbeef"
func newEndpointFromInputString(uri string) (*endpoint, error) {
	parsedURL, err := url.Parse(uri)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrInvalidInput, err)
	}
	var obfuscation string
	switch parsedURL.Scheme {
	case "openvpn":
		obfuscation = "openvpn"
	case "openvpn+obfs4":
		obfuscation = "obfs4"
	default:
		return nil, fmt.Errorf("%w: unknown scheme: %s", ErrInvalidInput, parsedURL.Scheme)
	}

	host := parsedURL.Hostname()
	if host == "" {
		return nil, fmt.Errorf("%w: expected host: %s", ErrInvalidInput, parsedURL.Host)
	}

	port := parsedURL.Port()
	if port == "" {
		return nil, fmt.Errorf("%w: expected port: %s", ErrInvalidInput, parsedURL.Port())
	}

	pathParts := strings.Split(parsedURL.Path, "/")
	if len(pathParts) != 3 {
		return nil, fmt.Errorf("%w: invalid path: %s (%d)", ErrInvalidInput, pathParts, len(pathParts))
	}
	transport := pathParts[1]
	if transport != "tcp" && transport != "udp" {
		return nil, fmt.Errorf("%w: invalid transport: %s", ErrInvalidInput, transport)
	}

	params := parsedURL.Query()
	provider := params.Get("provider")

	if provider == "" {
		return nil, fmt.Errorf("%w: please specify a provider as part of the input", ErrInvalidInput)
	}

	if provider != "riseup" {
		// because we are hardcoding at the moment. figure out a way to pass info for
		// arbitrary providers as options instead
		return nil, fmt.Errorf("%w: unknown provider: %s", ErrInvalidInput, provider)
	}

	endpoint := &endpoint{
		IPAddr:      host,
		Obfuscation: obfuscation,
		Port:        port,
		Protocol:    "openvpn",
		Provider:    provider,
		Transport:   transport,
	}
	return endpoint, nil
}

// String implements Stringer. This is a subset of the input URI scheme.
func (e *endpoint) String() string {
	var proto string
	if e.Obfuscation == "obfs4" {
		proto = e.Protocol + "+obfs4"
	} else {
		proto = e.Protocol
	}
	return fmt.Sprintf("%s://%s:%s/%s", proto, e.IPAddr, e.Port, e.Transport)
}

// AsInputURI is a string representation of this endpoint. It contains more information than the endpoint itself.
// TODO: redo with latest format
// openvpn://provider.corp/?address=1.1.1.1:1194&transport=tcp
func (e *endpoint) AsInputURI() string {
	provider := e.Provider
	if provider == "" {
		provider = "unknown"
	}
	i := fmt.Sprintf("%s/?provider=%s", e.String(), provider)
	return i
}

// endpointList is a list of endpoints.
type endpointList []*endpoint

// allEndpoints contains a subset of known endpoints to be used if no input is passed to the experiment.
// This is a hardcoded list for now, but the idea is that we can receive this from the check-in api in the future.
// In any case, having hardcoded endpoints is good as a fallback for the cases in which we cannot contact
// OONI's backend.
var allEndpoints = endpointList{
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

func isValidProvider(provider string) bool {
	_, ok := defaultOptionsByProvider[provider]
	return ok
}

// getVPNConfig gets a properly configured [*vpnconfig.Config] object for the given endpoint.
// To obtain that, we merge the endpoint specific configuration with base options.
// These base options are for the moment hardcoded. In the future we will want to be smarter
// about getting information for different providers.
func getVPNConfig(tracer *vpntracex.Tracer, endpoint *endpoint, creds *vpnconfig.OpenVPNOptions) (*vpnconfig.Config, error) {

	// TODO(ainghazal): use merge ability in vpnconfig.OpenVPNOptions merge (pending PR)

	provider := endpoint.Provider
	if !isValidProvider(provider) {
		return nil, fmt.Errorf("%w: unknown provider: %s", ErrInvalidInput, provider)
	}

	baseOptions := defaultOptionsByProvider[provider]

	cfg := vpnconfig.NewConfig(
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
		vpnconfig.WithHandshakeTracer(tracer))

	// TODO: validate options here and return an error.
	return cfg, nil
}

func extractBase64Blob(val string) (string, error) {
	s := strings.TrimPrefix(val, "base64:")
	if len(s) == len(val) {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, "missing prefix")
	}
	dec, err := base64.URLEncoding.DecodeString(strings.TrimSpace(s))
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrBadBase64Blob, err)
	}
	if len(dec) == 0 {
		return "", nil
	}
	return string(dec), nil
}
