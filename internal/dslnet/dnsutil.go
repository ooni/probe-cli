package dslnet

// dnsUtilReturn is a convenience function to return from [Getaddrinfo] and [DNSLookupUDP].
func dnsUtilReturn(query DNSQuery, addrs []string, err error) ([]Endpoint, error) {
	// handle error case
	if err != nil {
		return nil, err
	}

	// handle successful case
	outputs := []Endpoint{}
	for _, addr := range addrs {
		outputs = append(outputs, NewEndpoint(query.EndpointTemplate, query.Domain, addr))
	}
	return outputs, nil
}
