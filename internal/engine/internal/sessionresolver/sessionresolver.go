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
	ByteCounter *bytecounter.Counter // optional
	KVStore     KVStore              // mandatory
	Logger      Logger               // optional
	codec       codec
	mu          sync.Mutex
	once        sync.Once
	res         map[string]resolver
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
	r.maybeConfusion(state)
	defer r.writestate(state)
	const ewma = 0.9 // the last sample is very important
	me := multierror.New(ErrLookupHost)
	for _, e := range state {
		re, err := r.getresolver(e.URL)
		if err != nil {
			r.logger().Warnf("sessionresolver: getresolver: %s", err.Error())
			continue
		}
		addrs, err := r.timeLimitedLookup(ctx, re, hostname)
		if err == nil {
			r.logger().Infof("sessionresolver: %s... %v", e.URL, nil)
			e.Score = ewma*1.0 + (1-ewma)*e.Score // increase score
			return addrs, nil
		}
		r.logger().Warnf("sessionresolver: %s... %s", e.URL, err.Error())
		e.Score = ewma*0.0 + (1-ewma)*e.Score // decrease score
		me.Add(&errwrapper{error: err, URL: e.URL})
	}
	return nil, me
}

// maybeConfusion will rearrange the  first elements of the vector
// with low probability, so giving other resolvers a chance
// to run and show that they are also viable. We do not fully
// reorder the vector because that could lead to long runtimes.
func (r *Resolver) maybeConfusion(state []*resolverinfo) {
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	const confusion = 0.3
	if rng.Float64() >= confusion {
		return
	}
	switch len(state) {
	case 0, 1: // nothing to do
	case 2:
		state[0], state[1] = state[1], state[0]
	default:
		state[0], state[2] = state[2], state[0]
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
