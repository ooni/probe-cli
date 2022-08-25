package tlsmiddlebox

import "time"

// Config contains the experiment configuration.
type Config struct {
	// ResolverURL is the default DoH resolver
	ResolverURL string `ooni:"URL for DoH resolver"`

	// SNIPass is the SNI value we don't expect to be blocked
	SNIControl string `ooni:"the SNI value to cal"`

	// Delay is the delay between each iteration (in milliseconds).
	Delay int64 `ooni:"delay between consecutive iterations"`

	// Iterations is the default number of interations we trace
	MaxTTL int64 `ooni:"iterations is the number of iterations"`

	// SNI is the SNI value to use.
	SNI string `ooni:"the SNI value to use"`

	// ClientId is the client fingerprint to use
	ClientId int `ooni:"the ClientHello fingerprint to use"`
}

func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}

func (c Config) snicontrol() string {
	if c.SNIControl != "" {
		return c.SNIControl
	}
	return "example.com"
}

func (c Config) delay() time.Duration {
	if c.Delay > 0 {
		return time.Duration(c.Delay) * time.Millisecond
	}
	return 100 * time.Millisecond
}

func (c Config) maxttl() int64 {
	if c.MaxTTL > 0 {
		return c.MaxTTL
	}
	return 20
}

func (c Config) sni(address string) string {
	if c.SNI != "" {
		return c.SNI
	}
	return address
}

func (c Config) clientid() int {
	if c.ClientId > 0 {
		return c.ClientId
	}
	return 0
}
