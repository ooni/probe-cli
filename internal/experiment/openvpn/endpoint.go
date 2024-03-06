package openvpn

import "fmt"

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
