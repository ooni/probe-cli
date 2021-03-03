// Package sessionresolver contains the resolver used by the session. This
// resolver will try to figure out which is the best service for running
// domain name resolutions and will consistently use it.
//
// Occasionally this code will also swap the best resolver with other
// ~good resolvers to give them a chance to perform.
//
// The penalty/reward mechanism is strongly derivative, so the code should
// adapt ~quickly to changing network conditions. Occasionally, we will
// have longer resolutions when trying out other resolvers.
//
// At the beginning we randomize the known resolvers so that we do not
// have any preferential ordering. The initial resolutions may be slower
// if there are many issues with resolvers.
//
// The system resolver is given the lowest priority at the beginning
// but it will of course be the most popular resolver if anything else
// is failing us. (We will still occasionally probe for other working
// resolvers and increase their score on success.)
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
	res            map[string]childResolver
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

// LookupHost implements Resolver.LookupHost. This function returns a
// multierror.Union error on failure, so you can see individual errors
// and get a better picture of what's been going wrong.
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
