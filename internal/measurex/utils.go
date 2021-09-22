package measurex

//
// Utils
//
// This is where we put free functions.
//

// alpnForHTTPEndpoint selects the correct ALPN for an HTTP endpoint
// given the network. On failure, we return a nil list.
func alpnForHTTPEndpoint(network EndpointNetwork) []string {
	switch network {
	case NetworkQUIC:
		return []string{"h3"}
	case NetworkTCP:
		return []string{"h2", "http/1.1"}
	default:
		return nil
	}
}
