package tlsmiddlebox

//
// Config for the tlsmiddlebox experiment
//

import (
	"net/url"
	"time"
)

// Config contains the experiment configuration.
type Config struct {
	// ResolverURL is the default DoH resolver
	ResolverURL string `json:"resolver_url" ooni:"URL for DoH resolver"`

	// SNIPass is the SNI value we don't expect to be blocked
	SNIControl string `json:"sni_control" ooni:"control SNI value for testhelper"`

	// Delay is the delay between each iteration (in milliseconds).
	Delay int64 `json:"delay" ooni:"delay between consecutive iterations"`

	// MaxTTL is the default number of interations we trace
	MaxTTL int64 `json:"maximum_ttl" ooni:"maximum TTL value to iterate upto"`

	// TestHelper is the testhelper host for iterative tracing
	TestHelper string `json:"test_helper" ooni:"testhelper URL to use for tracing"`

	// ClientId is the client fingerprint to use
	ClientId int `json:"client_id" ooni:"ClientHello fingerprint to use"`

	// PrivacyMode is whether the user only wants to reveal AS and country information or city-level geolocation and /24 prefix information with respect to traceroutes
	PrivacyMode string `json:"privacy_mode" ooni:"privacy mode to use for traceroutes"`
}

// resolverURL returns the ResolverURL from the experiment's Config struct
func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}

// snicontrol returns the SNIControl from the experiment's Config struct
func (c Config) snicontrol() string {
	if c.SNIControl != "" {
		return c.SNIControl
	}
	return "example.com"
}

// delay returns the Delay from the experiment's Config struct
func (c Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return 100 * time.Millisecond
}

// maxttl returns the MaxTTL from the experiment's Config struct
func (c Config) maxttl() int64 {
	if c.MaxTTL > 0 {
		return c.MaxTTL
	}
	return 20
}

// testhelper returns the TestHelper from the experiment's Config struct
func (c Config) testhelper(address string) (URL *url.URL, err error) {
	// TODO(DecFox, bassosimone): We want to replace this with a generic input parser
	// Issue: https://github.com/ooni/probe/issues/2239
	if c.TestHelper != "" {
		return url.Parse(c.TestHelper)
	}
	URL = &url.URL{
		Host:   address,
		Scheme: "tlshandshake",
	}
	return
}

// clientid returns the ClientId from the experiment's Config struct
func (c Config) clientid() int {
	if c.ClientId > 0 {
		return c.ClientId
	}
	return 0
}

// privacymode returns the PrivacyMode from the experiment's Config struct
func (c Config) privacymode() string {
	if c.PrivacyMode != "" {
		return c.PrivacyMode
	}

	return "default"
}
