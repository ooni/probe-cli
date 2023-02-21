// Package inputparser contains code to parse experiments input.
package inputparser

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"reflect"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/idna"
)

// Config contains config for parsing experiments input. You MUST set
// the fields marked as MANDATORY otherwise Parse will fail.
type Config struct {
	// AcceptedSchemes is the list of accepted URL schemes. This field is
	// MANDATORY except when parsing endpoints where we do not need to
	// validate the scheme since we use DefaultScheme.
	AcceptedSchemes []string

	// AllowEndpoints OPTIONALLY tells the input parser to also
	// accept endpoints as experiment inputs.
	AllowEndpoints bool

	// DefaultScheme is the scheme to use when accepting endpoints,
	// which is MANDATORY iff AllowEndpoints is true.
	DefaultScheme string
}

// ErrEmptyDefaultScheme indicates that the default scheme is empty.
var ErrEmptyDefaultScheme = errors.New("inputparser: empty default scheme")

// ErrEmptyHostname indicates that the URL.Hostname() is empty.
var ErrEmptyHostname = errors.New("inputparser: empty URL.Hostname()")

// ErrIDNAToASCII indicates that we cannot convert IDNA to ASCII.
var ErrIDNAToASCII = errors.New("inputparser: cannot convert IDNA to ASCII")

// ErrInvalidEndpoint indicates that we are not parsing a valid endpoint.
var ErrInvalidEndpoint = errors.New("inputparser: invalid endpoint")

// ErrURLParse indicates that we could not parse the URL.
var ErrURLParse = errors.New("inputparser: cannot parse URL")

// ErrUnsupportedScheme indicates that we do not support the given URL.Scheme.
var ErrUnsupportedScheme = errors.New("inputparser: unsupported URL.Scheme")

// Parse parses the experiment input using the given config and returns
// to the caller either the resulting URL or an error.
func Parse(config *Config, input model.MeasurementTarget) (*url.URL, error) {
	runtimex.Assert(config != nil, "passed nil config")
	runtimex.Assert(input != "", "passed empty input")

	// Attempt to parse the input as an URL.
	URL, err := url.Parse(string(input))
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrURLParse, err.Error())
	}

	// Reject empty URL.Hostname().
	if URL.Hostname() == "" {
		// If we are not allowed to parse endpoints, just emit an error.
		if !config.AllowEndpoints {
			return nil, ErrEmptyHostname
		}

		// Check whether we could interpret the URL as an endpoint.
		URL, err = maybeEndpointToURL(config, URL)
		if err != nil {
			return nil, err
		}
		// Fallthrough on success.
	}

	// Reject schemes that are not allowed for this experiment.
	if !isSchemeOK(config, URL) {
		return nil, ErrUnsupportedScheme
	}

	// Possibly rewrite the URL.Host to be in punycode.
	return maybeConvertHostnameToASCII(URL)
}

// maybeEndpointToURL takes in input an already parsed URL and returns
// in output either a new URL containing an endpoint with the configured
// default scheme or an error. For example, given this input:
//
//	&url.URL{Scheme:"example.com",Opaque:"80"}
//
// and `http` as the config.DefaultScheme, this function would return:
//
//	&url.URL{Scheme:"http",Host:"example.com:80"}
//
// See https://go.dev/play/p/Rk5pS_zGY5U for additional information on how
// URL.Parse will parse "example.com:80" and other endpoints.
func maybeEndpointToURL(config *Config, URL *url.URL) (*url.URL, error) {
	// Make sure the parsing result is exactly what we expected.
	expect := &url.URL{
		Scheme: URL.Scheme,
		Opaque: URL.Opaque,
	}
	if !reflect.DeepEqual(URL, expect) {
		return nil, ErrInvalidEndpoint
	}

	// Make sure we actually have a valid default scheme.
	if config.DefaultScheme == "" {
		return nil, ErrEmptyDefaultScheme
	}

	// Rewrite the URL to contain the endpoint.
	URL = &url.URL{
		Scheme: config.DefaultScheme,
		Host:   net.JoinHostPort(expect.Scheme, expect.Opaque),
	}
	return URL, nil
}

// maybeConvertHostnameToASCII takes in input a URL and converts
// the URL.Host to become ASCII. This function MUTATES the input URL
// in place and returns either the mutated URL or an error.
func maybeConvertHostnameToASCII(URL *url.URL) (*url.URL, error) {
	hostname := URL.Hostname()

	// Obtain an ASCII representation of the URL.Hostname().
	asciiHostname, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrIDNAToASCII, err.Error())
	}

	// Possibly rewrite the URL.Host to be in punycode.
	if asciiHostname != hostname {
		if port := URL.Port(); port != "" {
			URL.Host = net.JoinHostPort(asciiHostname, port)
		} else {
			URL.Host = asciiHostname
		}
	}

	// Return the parsed URL to the caller.
	return URL, nil
}

// isSchemeOK indicates whether the given URL scheme is OK.
func isSchemeOK(config *Config, URL *url.URL) bool {
	for _, scheme := range config.AcceptedSchemes {
		if URL.Scheme == scheme {
			return true
		}
	}
	// We don't need to provide AcceptedSchemes when ONLY parsing endpoints.
	return config.AllowEndpoints && URL.Scheme == config.DefaultScheme
}
