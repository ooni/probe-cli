package enginenetx

//
// static policy - the possibility of loading a static policy from a JSON
// document named `httpsdialerstatic.conf` in $OONI_HOME/engine that contains
// a specific policy for TLS dialing for specific endpoints.
//
// This policy helps a lot with exploration and experimentation.
//

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/hujsonx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// staticPolicy is an [httpsDialerPolicy] incorporating verbatim
// a static policy loaded from the engine's key-value store.
//
// This policy is very useful for exploration and experimentation.
type staticPolicy struct {
	// Fallback is the fallback policy in case the static one does not
	// contain a rule for a specific domain.
	Fallback httpsDialerPolicy

	// Root is the root of the statically loaded policy.
	Root *staticPolicyRoot
}

// staticPolicyKey is the kvstore key used to retrieve the static policy.
const staticPolicyKey = "httpsdialerstatic.conf"

// errStaticPolicyWrongVersion means that the static policy document has the wrong version number.
var errStaticPolicyWrongVersion = errors.New("wrong static policy version")

// newStaticPolicy attempts to constructs a static policy using a given fallback
// policy and either returns a good policy or an error. The typical error case is the one
// in which there's no httpsDialerStaticPolicyKey in the key-value store.
func newStaticPolicy(
	kvStore model.KeyValueStore, fallback httpsDialerPolicy) (*staticPolicy, error) {
	// attempt to read the static policy bytes from the kvstore
	data, err := kvStore.Get(staticPolicyKey)
	if err != nil {
		return nil, err
	}

	// attempt to parse the static policy using human-readable JSON
	var root staticPolicyRoot
	if err := hujsonx.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	// make sure the version is OK
	if root.Version != staticPolicyVersion {
		err := fmt.Errorf(
			"%s: %w: expected=%d got=%d",
			staticPolicyKey,
			errStaticPolicyWrongVersion,
			staticPolicyVersion,
			root.Version,
		)
		return nil, err
	}

	out := &staticPolicy{
		Fallback: fallback,
		Root:     &root,
	}
	return out, nil
}

// staticPolicyVersion is the current version of the static policy file.
const staticPolicyVersion = 3

// staticPolicyRoot is the root of a statically loaded policy.
type staticPolicyRoot struct {
	// DomainEndpoints maps each domain endpoint to its policies.
	DomainEndpoints map[string][]*httpsDialerTactic

	// Version is the data structure version.
	Version int
}

var _ httpsDialerPolicy = &staticPolicy{}

// LookupTactics implements httpsDialerPolicy.
func (ldp *staticPolicy) LookupTactics(
	ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	tactics, found := ldp.Root.DomainEndpoints[net.JoinHostPort(domain, port)]
	if !found {
		return ldp.Fallback.LookupTactics(ctx, domain, port)
	}

	out := make(chan *httpsDialerTactic)
	go func() {
		defer close(out)
		for _, tactic := range tactics {
			out <- tactic
		}
	}()
	return out
}
