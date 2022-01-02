package sessionresolver

import (
	"math/rand"
	"strings"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
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
	url: "https://dns.quad9.net/dns-query",
}, {
	url: "https://doh.powerdns.org/",
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

// byteCounter returns the configured byteCounter or a default
func (r *Resolver) byteCounter() *bytecounter.Counter {
	if r.ByteCounter != nil {
		return r.ByteCounter
	}
	return bytecounter.New()
}

// logger returns the configured logger or a default
func (r *Resolver) logger() model.Logger {
	if r.Logger != nil {
		return r.Logger
	}
	return log.Log
}

// newresolver creates a new resolver with the given config and URL. This is
// where we expand http3 to https and set the h3 options.
func (r *Resolver) newresolver(URL string) (childResolver, error) {
	h3 := strings.HasPrefix(URL, "http3://")
	if h3 {
		URL = strings.Replace(URL, "http3://", "https://", 1)
	}
	return r.clientmaker().Make(netx.Config{
		BogonIsError: true,
		ByteCounter:  r.byteCounter(),
		HTTP3Enabled: h3,
		Logger:       r.logger(),
		ProxyURL:     r.ProxyURL,
	}, URL)
}

// getresolver returns a resolver with the given URL. This function caches
// already allocated resolvers so we only allocate them once.
func (r *Resolver) getresolver(URL string) (childResolver, error) {
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
		r.res = make(map[string]childResolver)
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
