package sessionresolver

//
// Code for creating a new child resolver
//

import (
	"math/rand"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// resolvemaker contains rules for making a resolver.
type resolvermaker struct {
	url   string
	score float64
}

// systemResolverURL is the URL of the system resolver.
const systemResolverURL = "system:///"

// allmakers contains all the makers in a list. We use the http3
// prefix to indicate we wanna use http3. The code will translate
// this to https and set the proper netx options.
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
	url: systemResolverURL,
}, {
	url: "https://mozilla.cloudflare-dns.com/dns-query",
}, {
	url: "http3://mozilla.cloudflare-dns.com/dns-query",
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

// logger returns the configured logger or a default
func (r *Resolver) logger() model.Logger {
	return model.ValidLoggerOrDefault(r.Logger)
}

// newChildResolver creates a new child model.Resolver.
func (r *Resolver) newChildResolver(h3 bool, URL string) (model.Resolver, error) {
	if r.newChildResolverFn != nil {
		return r.newChildResolverFn(h3, URL)
	}
	return netx.NewDNSClient(netx.Config{
		BogonIsError: true,
		ByteCounter:  r.ByteCounter, // nil is handled by netx
		HTTP3Enabled: h3,
		Logger:       r.logger(),
		ProxyURL:     r.ProxyURL,
	}, URL)
}

// newresolver creates a new resolver with the given config and URL. This is
// where we expand http3 to https and set the h3 options.
func (r *Resolver) newresolver(URL string) (model.Resolver, error) {
	h3 := strings.HasPrefix(URL, "http3://")
	if h3 {
		URL = strings.Replace(URL, "http3://", "https://", 1)
	}
	return r.newChildResolver(h3, URL)
}

// getresolver returns a resolver with the given URL. This function caches
// already allocated resolvers so we only allocate them once.
func (r *Resolver) getresolver(URL string) (model.Resolver, error) {
	defer r.mu.Unlock()
	r.mu.Lock()
	if re, found := r.res[URL]; found {
		return re, nil // already created
	}
	re, err := r.newresolver(URL)
	if err != nil {
		return nil, err // config err?
	}
	if r.res == nil {
		r.res = make(map[string]model.Resolver)
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
