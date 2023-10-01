package pdsl

import "net"

// Endpoint is a string containing a TCP/UDP endpoint.
type Endpoint string

// MakeEndpointsForPort returns a [Filter] that either passes errors through
// or converts an [IPAddr] to an [Endpoint] using the given port.
func MakeEndpointsForPort(port string) Filter[Result[IPAddr], Result[Endpoint]] {
	return func(inputs <-chan Result[IPAddr]) <-chan Result[Endpoint] {
		outputs := make(chan Result[Endpoint])

		go func() {
			defer close(outputs)
			for input := range inputs {
				if err := input.Err; err != nil {
					outputs <- NewResultError[Endpoint](err)
					continue
				}
				outputs <- NewResultValue(Endpoint(net.JoinHostPort(string(input.Value), port)))
			}
		}()

		return outputs
	}
}
