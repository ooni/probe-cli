// Package sessionresolver contains the resolver used by the session. This
// resolver will try to figure out which is the best service for running
// domain name resolutions and will consistently use it.
package sessionresolver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/multierror"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
)

// Resolver is the session resolver. You should create an instance of
// this structure and use it in session.go.
type Resolver struct {
	ByteCounter    *bytecounter.Counter // optional
	KVStore        KVStore              // optional
	Logger         Logger               // optional
	codec          codec
	dnsClientMaker dnsclientmaker
	mu             sync.Mutex
	once           sync.Once
	res            map[string]resolver
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

// ErrLookupHost indicates that LookupHost failed.
var ErrLookupHost = errors.New("sessionresolver: LookupHost failed")

// LookupHost implements Resolver.LookupHost.
func (r *Resolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	state := r.readstatedefault()
	r.maybeConfusion(state, time.Now().UnixNano())
	defer r.writestate(state)
	me := multierror.New(ErrLookupHost)
	for _, e := range state {
		addrs, err := r.lookupHost(ctx, e, hostname)
		if err == nil {
			return addrs, nil
		}
		me.Add(&errwrapper{error: err, URL: e.URL})
	}
	return nil, me
}

func (r *Resolver) lookupHost(ctx context.Context, ri *resolverinfo, hostname string) ([]string, error) {
	const ewma = 0.9 // the last sample is very important
	re, err := r.getresolver(ri.URL)
	if err != nil {
		r.logger().Warnf("sessionresolver: getresolver: %s", err.Error())
		ri.Score = 0 // this is a hard error
		return nil, err
	}
	addrs, err := r.timeLimitedLookup(ctx, re, hostname)
	if err == nil {
		r.logger().Infof("sessionresolver: %s... %v", ri.URL, nil)
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
