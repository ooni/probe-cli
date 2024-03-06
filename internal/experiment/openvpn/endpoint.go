package openvpn

// endpoint is a single endpoint to be probed.
type endpoint struct {
	// Provider is a unique label identifying the provider maintaining this endpoint.
	Provider string

	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// Port is the Port for this endpoint.
	Port string

	// Transport is the underlying transport used for this endpoint. Valid transports are `tcp` and `udp`.
	Transport string
}

// allEndpoints contains a subset of known endpoints to be used if no input is passed to the experiment.
var allEndpoints = []endpoint{
	{
		Provider:  "riseup",
		IPAddr:    "185.220.103.11",
		Port:      "1194",
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
