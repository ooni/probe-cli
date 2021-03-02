package sessionresolver

import (
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
)

// resolvemaker contains rules for making a resolver.
type resolvermaker struct {
	url   string
	score float64
}

// systemResolverURL is the URL of the system resolver.
const systemResolverURL = "system:///"

// allmakers contains all the makers in a list.
var allmakers = []*resolvermaker{{
	url: "https://cloudflare-dns.com/dns-query",
}, {
	url: "http3://cloudflare-dns.com/dns-query",
}, {
	url: "https://dns.google/dns-query",
}, {
	url: "http3://dns.google/dns-query",
}, {
	url: "https://dns.quad9.net/dns-query",
}, {
	url: "http3://dns.quad9.net/dns-query",
}, {
	url: "https://doh.powerdns.org/",
}, {
	url: systemResolverURL,
}}

// allbyurl contains all the resolvermakers by URL
var allbyurl map[string]*resolvermaker

// init fills allbyname and gives a nonzero initial score
// to all resolvers except for the system resolver. We set
// the system resolver score to zero, so that it's less
// likely than other resolvers in this list.
func init() {
	allbyurl = make(map[string]*resolvermaker)
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	for _, e := range allmakers {
		allbyurl[e.url] = e
		if e.url != systemResolverURL {
			e.score = rng.Float64()
		}
	}
}

// byteCounter returns the configured byteCounter or a default
func (c *Config) byteCounter() *bytecounter.Counter {
	if c.ByteCounter != nil {
		return c.ByteCounter
	}
	return bytecounter.New()
}

// logger returns the configured logger or a default
func (c *Config) logger() Logger {
	if c.Logger != nil {
		return c.Logger
	}
	return log.Log
}

// errNoResolver indicates that a resolver does not exist
var errNoResolver = errors.New("no such resolver")

// newresolver creates a new resolver with the given config and URL
func (r *Resolver) newresolver(config *Config, URL string) (resolver, error) {
	e, found := allbyurl[URL]
	if !found {
		return nil, fmt.Errorf("%w: %s", errNoResolver, URL)
	}
	h3 := strings.HasSuffix(URL, "http3")
	if h3 {
		strings.Replace(URL, "http3://", "https://", 1)
	}
	return netx.NewDNSClientWithOverrides(netx.Config{
		BogonIsError: true,
		ByteCounter:  config.byteCounter(),
		HTTP3Enabled: h3,
		Logger:       config.logger(),
	}, e.url, "", "", "")
}

// getresolver returns a resolver with the given URL. This function caches
// already allocated resolvers so we only allocate them once.
func (r *Resolver) getresolver(config *Config, URL string) (resolver, error) {
	defer r.mu.Unlock()
	r.mu.Lock()
	if re, found := r.res[URL]; found == true {
		return re, nil
	}
	re, err := r.newresolver(config, URL)
	if err != nil {
		return nil, err
	}
	if r.res == nil {
		r.res = make(map[string]resolver)
	}
	r.res[URL] = re
	return re, nil
}

// closeall closes the cached resolvers.
func (r *Resolver) closeall() {
	defer r.mu.Unlock()
	r.mu.Lock()
	for _, re := range r.res {
		re.CloseIdleConnections()
	}
	r.res = nil
}
