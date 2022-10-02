package echcheck

// Config contains the experiment config.
type Config struct {
	// ResolverURL is the default DoH resolver
	ResolverURL string `ooni:"URL for DoH resolver"`
}

func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return "https://mozilla.cloudflare-dns.com/dns-query"
}
