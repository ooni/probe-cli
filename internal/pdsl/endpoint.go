package pdsl

import "net"

// Endpoint is a string containing a TCP/UDP endpoint (e.g., 8.8.8.8:443, [::1]).
type Endpoint string

// MakeEndpointsForPort returns a [Filter] that attemps to make [Endpoint] from [IPAddr].
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
