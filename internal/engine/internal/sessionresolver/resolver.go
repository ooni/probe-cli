package sessionresolver

//
// Implementation of Resolver
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/multierror"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Resolver is the session resolver. Resolver will try to use
// a bunch of DoT/DoH resolvers before falling back to the
// system resolver. The relative priorities of the resolver
// are stored onto the KVStore such that we can remember them
// and therefore we can generally give preference to underlying
// DoT/DoH resolvers that work better.
//
// Make sure you fill the mandatory fields (indicated below)
// before using this data structure.
//
// You MUST NOT modify public fields of this structure once it
// has been created, because that MAY lead to data races.
type Resolver struct {
	// ByteCounter is the OPTIONAL byte counter. It will count
	// the bytes used by any child resolver except for the
	// system resolver, whose bytes ARE NOT counted. If this
	// field is not set, then we won't count the bytes.
	ByteCounter *bytecounter.Counter

	// KVStore is the MANDATORY key-value store where you
	// want us to write statistics about which resolver is
	// working better in your network.
	KVStore model.KeyValueStore

	// Logger is the OPTIONAL logger you want us to use
	// to emit log messages.
	Logger model.Logger

	// ProxyURL is the OPTIONAL URL of the socks5 proxy
	// we should be using. If not set, then we WON'T use
	// any proxy. If set, then we WON'T use any http3
	// based resolvers and we WON'T use the system resolver.
	ProxyURL *url.URL

	// jsonCodec is the OPTIONAL JSON Codec to use. If not set,
	// we will construct a default codec.
	jsonCodec jsonCodec

	// dnsClientMaker is the OPTIONAL dnsclientmaker to
	// use. If not set, we will use the default.
	dnsClientMaker dnsclientmaker

	// mu provides synchronisation of internal fields.
	mu sync.Mutex

	// once ensures that CloseIdleConnection is
	// run just once.
	once sync.Once

	// res maps a URL to a child resolver. We will
	// construct child resolvers just once and we
	// will track them into this field.
	res map[string]model.Resolver
}

// CloseIdleConnections closes the idle connections, if any. This
// function is guaranteed to be idempotent.
func (r *Resolver) CloseIdleConnections() {
	r.once.Do(r.closeall)
}

// Stats returns stats about the session resolver.
func (r *Resolver) Stats() string {
	data, err := json.Marshal(r.readstatedefault())
	runtimex.PanicOnError(err, "json.Marshal should not fail here")
	return fmt.Sprintf("sessionresolver: %s", string(data))
}

// errLookupNotImplemented indicates a given lookup type is not implemented.
var errLookupNotImplemented = errors.New("sessionresolver: lookup not implemented")

// LookupHTTPS implements Resolver.LookupHTTPS.
func (r *Resolver) LookupHTTPS(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, errLookupNotImplemented
}

// LookupNS implements Resolver.LookupNS.
func (r *Resolver) LookupNS(ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, errLookupNotImplemented
}

// ErrLookupHost indicates that LookupHost failed.
var ErrLookupHost = errors.New("sessionresolver: LookupHost failed")

// LookupHost implements Resolver.LookupHost. This function returns a
// multierror.Union error on failure, so you can see individual errors
// and get a better picture of what's been going wrong.
func (r *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	state := r.readstatedefault()
	r.maybeConfusion(state, time.Now().UnixNano())
	defer r.writestate(state)
	me := multierror.New(ErrLookupHost)
	for _, e := range state {
		if r.ProxyURL != nil && r.shouldSkipWithProxy(e) {
			r.logger().Infof("sessionresolver: skipping with proxy: %+v", e)
			continue // we cannot proxy this URL so ignore it
		}
		addrs, err := r.lookupHost(ctx, e, hostname)
		if err == nil {
			return addrs, nil
		}
		me.Add(newErrWrapper(err, e.URL))
	}
	return nil, me
}

func (r *Resolver) shouldSkipWithProxy(e *resolverinfo) bool {
	URL, err := url.Parse(e.URL)
	if err != nil {
		return true // please skip
	}
	switch URL.Scheme {
	case "https", "dot", "tcp":
		return false // we can handle this
	default:
		return true // please skip
	}
}

func (r *Resolver) lookupHost(ctx context.Context, ri *resolverinfo, hostname string) ([]string, error) {
	const ewma = 0.9 // the last sample is very important
	re, err := r.getresolver(ri.URL)
	if err != nil {
		r.logger().Warnf("sessionresolver: getresolver: %s", err.Error())
		ri.Score = 0 // this is a hard error
		return nil, err
	}
	addrs, err := timeLimitedLookup(ctx, re, hostname)
	if err == nil {
		r.logger().Infof("sessionresolver: %s... %v", ri.URL, model.ErrorToStringOrOK(nil))
		ri.Score = ewma*1.0 + (1-ewma)*ri.Score // increase score
		return addrs, nil
	}
	r.logger().Warnf("sessionresolver: %s... %s", ri.URL, err.Error())
	ri.Score = ewma*0.0 + (1-ewma)*ri.Score // decrease score
	return nil, err
}

// maybeConfusion will rearrange the  first elements of the vector
// with low probability, so giving other resolvers a chance
// to run and show that they are also viable. We do not fully
// reorder the vector because that could lead to long runtimes.
//
// The return value is only meaningful for testing.
func (r *Resolver) maybeConfusion(state []*resolverinfo, seed int64) int {
	rng := rand.New(rand.NewSource(seed))
	const confusion = 0.3
	if rng.Float64() >= confusion {
		return -1
	}
	switch len(state) {
	case 0, 1: // nothing to do
		return 0
	case 2:
		state[0], state[1] = state[1], state[0]
		return 2
	default:
		state[0], state[2] = state[2], state[0]
		return 3
	}
}

// Network implements Resolver.Network.
func (r *Resolver) Network() string {
	return "sessionresolver"
}

// Address implements Resolver.Address.
func (r *Resolver) Address() string {
	return ""
}
