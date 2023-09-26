package enginenetx

//
// user policy - the possibility of loading a user policy from a JSON
// document named `httpsdialer.conf` in $OONI_HOME/engine that contains
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

// userPolicy is an [httpsDialerPolicy] incorporating verbatim
// a user policy loaded from the engine's key-value store.
//
// This policy is very useful for exploration and experimentation.
type userPolicy struct {
	// Fallback is the fallback policy in case the user one does not
	// contain a rule for a specific domain.
	Fallback httpsDialerPolicy

	// Root is the root of the user policy loaded from disk.
	Root *userPolicyRoot
}

// userPolicyKey is the kvstore key used to retrieve the user policy.
const userPolicyKey = "httpsdialer.conf"

// errUserPolicyWrongVersion means that the user policy document has the wrong version number.
var errUserPolicyWrongVersion = errors.New("wrong user policy version")

// newUserPolicy attempts to constructs a user policy using a given fallback
// policy and either returns a good policy or an error. The typical error case is the one
// in which there's no httpsDialerUserPolicyKey in the key-value store.
func newUserPolicy(
	kvStore model.KeyValueStore, fallback httpsDialerPolicy) (*userPolicy, error) {
	// attempt to read the user policy bytes from the kvstore
	data, err := kvStore.Get(userPolicyKey)
	if err != nil {
		return nil, err
	}

	// attempt to parse the user policy using human-readable JSON
	var root userPolicyRoot
	if err := hujsonx.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	// make sure the version is OK
	if root.Version != userPolicyVersion {
		err := fmt.Errorf(
			"%s: %w: expected=%d got=%d",
			userPolicyKey,
			errUserPolicyWrongVersion,
			userPolicyVersion,
			root.Version,
		)
		return nil, err
	}

	out := &userPolicy{
		Fallback: fallback,
		Root:     &root,
	}
	return out, nil
}

// userPolicyVersion is the current version of the user policy file.
const userPolicyVersion = 3

// userPolicyRoot is the root of the user policy.
type userPolicyRoot struct {
	// DomainEndpoints maps each domain endpoint to its policies.
	DomainEndpoints map[string][]*httpsDialerTactic

	// Version is the data structure version.
	Version int
}

var _ httpsDialerPolicy = &userPolicy{}

// LookupTactics implements httpsDialerPolicy.
func (ldp *userPolicy) LookupTactics(
	ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	// check whether an entry exists in the user-provided map, which MAY be nil
	// if/when the user has chosen their policy to be as such
	tactics, found := ldp.Root.DomainEndpoints[net.JoinHostPort(domain, port)]
	if !found {
		return ldp.Fallback.LookupTactics(ctx, domain, port)
	}

	// note that we also need to fallback when the tactics contains an empty list
	// or a list that only contains nil entries
	tactics = userPolicyRemoveNilEntries(tactics)
	if len(tactics) <= 0 {
		return ldp.Fallback.LookupTactics(ctx, domain, port)
	}

	// emit the resuults, which may possibly be empty
	out := make(chan *httpsDialerTactic)
	go func() {
		defer close(out) // let the caller know we're done
		for _, tactic := range tactics {
			out <- tactic
		}
	}()
	return out
}

func userPolicyRemoveNilEntries(input []*httpsDialerTactic) (output []*httpsDialerTactic) {
	for _, entry := range input {
		if entry != nil {
			output = append(output, entry)
		}
	}
	return
}
