package measurexlite

//
// Input parsing for experiments
//

import (
	"errors"
	"net"
	"net/url"
)

var (
	// ErrInvalidConfiguration indiactes that the parser configuration is invalid
	ErrInvalidConfiguration = errors.New("invalid configuration for InputParser.AllowEndpoints")

	// ErrInvalidInput indicates that the input is invalid
	ErrInvalidInput = errors.New("input is invalid")

	// ErrInvalidScheme indicates that the scheme is invalid
	ErrInvalidScheme = errors.New("parsed scheme is invalid")
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
	// put this check at top-level such that we always see if the parser is misconfigured
	if ip.AllowEndpoints && ip.DefaultScheme == "" {
		return nil, ErrInvalidConfiguration
	}
	URL, err := url.ParseRequestURI(input)
	if err != nil || URL.Host == "" {
		return ip.maybeAllowEndpoints(URL)
	}
	for _, scheme := range ip.AcceptedSchemes {
		if URL.Scheme == scheme {
			return URL, nil
		}
	}
	return nil, ErrInvalidScheme
}

// Conditionally allows endpoints when ip.AllowEndpoints is true.
func (ip *InputParser) maybeAllowEndpoints(URL *url.URL) (*url.URL, error) {
	if !ip.AllowEndpoints {
		return nil, ErrInvalidConfiguration
	}
	if URL != nil && URL.Scheme != "" && URL.Opaque != "" && URL.User == nil &&
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
	return nil, ErrInvalidInput
}
