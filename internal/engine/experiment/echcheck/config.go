package echcheck

const (
	defaultResolver = "https://mozilla.cloudflare-dns.com/dns-query"
)

// Config contains the experiment config.
type Config struct {
	// ResolverURL is the default DoH resolver
	ResolverURL string `ooni:"URL for DoH resolver"`
}

func (c Config) resolverURL() string {
	if c.ResolverURL != "" {
		return c.ResolverURL
	}
	return defaultResolver
}
