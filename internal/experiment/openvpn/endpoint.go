package openvpn

import (
	"fmt"

	"github.com/ooni/minivpn/pkg/config"
	vpnconfig "github.com/ooni/minivpn/pkg/config"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

// endpoint is a single endpoint to be probed.
type endpoint struct {
	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// Port is the Port for this endpoint.
	Port string

	// Protocol is the tunneling protocol (openvpn, openvpn+obfs4).
	Protocol string

	// Provider is a unique label identifying the provider maintaining this endpoint.
	Provider string

	// Transport is the underlying transport used for this endpoint. Valid transports are `tcp` and `udp`.
	Transport string
}

func (e *endpoint) String() string {
	return fmt.Sprintf("%s://%s:%s/%s", e.Protocol, e.IPAddr, e.Port, e.Transport)
}

// allEndpoints contains a subset of known endpoints to be used if no input is passed to the experiment.
var allEndpoints = []endpoint{
	{
		Provider:  "riseup",
		IPAddr:    "185.220.103.11",
		Port:      "1194",
		Protocol:  "openvpn",
		Transport: "tcp",
	},
}

// sampleRandomEndpoint is a placeholder for a proper sampling function.
func sampleRandomEndpoint(all []endpoint) endpoint {
	// chosen by fair dice roll
	// guaranteed to be random
	// https://xkcd.com/221/
	return all[0]
}

var defaultOptionsByProvider = map[string]*vpnconfig.OpenVPNOptions{
	"riseup": {
		Cipher: "AES-256-GCM",
		Auth:   "SHA512",
		CA:     []byte{},
		Key:    []byte{},
		Cert:   []byte{},
	},
}

// getVPNConfig gets a properly configured [*vpnconfig.Config] object for the given endpoint.
// To obtain that, we merge the endpoint specific configuration with base options.
// These base options are for the moment hardcoded. In the future we will want to be smarter
// about getting information for different providers.
func getVPNConfig(tracer *vpntracex.Tracer, endpoint *endpoint) (*vpnconfig.Config, error) {
	// TODO(ainghazal): use options merge
	provider := endpoint.Provider
	// TODO(ainghazal): return error if provider unknown. we're in the happy path for now.
	baseOptions := defaultOptionsByProvider[provider]

	cfg := vpnconfig.NewConfig(
		vpnconfig.WithOpenVPNOptions(
			&vpnconfig.OpenVPNOptions{
				Remote: endpoint.IPAddr,
				Port:   endpoint.Port,
				Proto:  vpnconfig.Proto(endpoint.Transport),

				// options coming from the default values,
				// to be changed by check-in API info in the future.
				CA:     baseOptions.CA,
				Cert:   baseOptions.Cert,
				Key:    baseOptions.Key,
				Cipher: baseOptions.Cipher,
				Auth:   baseOptions.Auth,
			},
		),
		config.WithHandshakeTracer(tracer))
	// TODO: validate here and return an error, maybe.
	return cfg, nil
}
