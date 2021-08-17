package websteps

import (
	"context"
	"errors"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// InitialChecks is the first step of the test helper algorithm. We
// make sure we can parse the URL, we handle the scheme, and the domain
// name inside the URL's authority is valid.

// Errors returned by Preresolve.
var (
	// ErrInvalidURL indicates that the URL is invalid.
	ErrInvalidURL = errors.New("the URL is invalid")

	// ErrUnsupportedScheme indicates that we don't support the scheme.
	ErrUnsupportedScheme = errors.New("unsupported scheme")

	// ErrNoSuchHost indicates that the DNS resolution failed.
	ErrNoSuchHost = errors.New("no such host")
)

// InitChecker is the interface responsible for running InitialChecks.
type InitChecker interface {
	InitialChecks(URL string) (*url.URL, error)
}

// DefaultInitChecker is the default InitChecker.
type DefaultInitChecker struct {
	resolver netxlite.Resolver
}

// InitialChecks checks whether the URL is valid and whether the
// domain inside the URL is an existing one. If these preliminary
// checks fail, there's no point in continuing.
// If they succeed, InitialChecks returns the URL, if not an error.
func (i *DefaultInitChecker) InitialChecks(URL string) (*url.URL, error) {
	parsed, err := url.Parse(URL)
	if err != nil {
		return nil, ErrInvalidURL
	}
	switch parsed.Scheme {
	case "http", "https":
	default:
		return nil, ErrUnsupportedScheme
	}
	// Assumptions:
	//
	// 1. the resolver will cache the resolution for later
	//
	// 2. an IP address does not cause an error because we are using
	// a resolve that behaves like getaddrinfo
	if _, err := i.resolver.LookupHost(context.Background(), parsed.Hostname()); err != nil {
		return nil, ErrNoSuchHost
	}
	return parsed, nil
}
