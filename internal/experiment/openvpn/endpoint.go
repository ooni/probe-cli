package openvpn

// Endpoint is a single endpoint to be probed.
type Endpoint struct {
	// Provider is a unique label identifying the provider maintaining this endpoint.
	Provider string

	// IPAddr is the IP Address for this endpoint.
	IPAddr string

	// Port is the Port for this endpoint.
	Port string

	// Transport is the underlying transport used for this endpoint. Valid transports are `tcp` and `udp`.
	Transport string
}
