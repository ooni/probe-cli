package webconnectivity

//
// Input parsing
//

import (
	"errors"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// InputParser helps to print the experiment's input.
type InputParser struct {
	// List of accepted URL schemes.
	AcceptedSchemes []string

	// Whether to allow endpoints in input.
	AllowEndpoints bool

	// The default scheme to use if AllowEndpoints == true.
	DefaultScheme string
}

// Parse parses the experiment input and returns the resulting URL.
func (ip *InputParser) Parse(input string) (*url.URL, error) {
	// put this check at top-level such that we always see the crash if needed
	runtimex.PanicIfTrue(
		ip.AllowEndpoints && ip.DefaultScheme == "",
		"invalid configuration for InputParser.AllowEndpoints == true",
	)
	URL, err := url.Parse(input)
	if err != nil {
		return ip.maybeAllowEndpoints(URL, err)
	}
	for _, scheme := range ip.AcceptedSchemes {
		if URL.Scheme == scheme {
			// TODO: here you may want to perform additional parsing
			return URL, nil
		}
	}
	return nil, errors.New("cannot parse input")
}

// Conditionally allows endpooints when ip.AllowEndpoints is true.
func (ip *InputParser) maybeAllowEndpoints(URL *url.URL, err error) (*url.URL, error) {
	runtimex.PanicIfNil(err, "expected to be called with a non-nil error")
	if ip.AllowEndpoints && URL.Scheme != "" && URL.Opaque != "" && URL.User == nil &&
		URL.Host == "" && URL.Path == "" && URL.RawPath == "" &&
		URL.RawQuery == "" && URL.Fragment == "" && URL.RawFragment == "" {
		// See https://go.dev/play/p/Rk5pS_zGY5U
		//
		// Note that we know that `ip.DefaultScheme != ""` from the above runtime check.
		out := &url.URL{
			Scheme: ip.DefaultScheme,
			Host:   net.JoinHostPort(URL.Scheme, URL.Opaque),
		}
		return out, nil
	}
	return nil, err
}
