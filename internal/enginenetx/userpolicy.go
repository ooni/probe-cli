package enginenetx

//
// user policy - the possibility of loading a user policy from a JSON
// document named `bridges.conf` in $OONI_HOME/engine that contains
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

// userPolicyV2 is an [httpsDialerPolicy] incorporating verbatim
// a user policy loaded from the engine's key-value store.
//
// This policy is very useful for exploration and experimentation.
//
// This is v2 of the userPolicy because the previous implementation
// incorporated mixing logic, while now the mixing happens outside
// of this policy, thus giving us much more flexibility.
type userPolicyV2 struct {
	// Root is the root of the user policy loaded from disk.
	Root *userPolicyRoot
}

// newUserPolicyV2 attempts to constructs a user policy. The typical error case is the one
// in which there's no httpsDialerUserPolicyKey in the key-value store.
func newUserPolicyV2(kvStore model.KeyValueStore) (*userPolicyV2, error) {
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

	out := &userPolicyV2{Root: &root}
	return out, nil
}

var _ httpsDialerPolicy = &userPolicyV2{}

// LookupTactics implements httpsDialerPolicy.
func (ldp *userPolicyV2) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	// create the output channel
	out := make(chan *httpsDialerTactic)

	go func() {
		// make sure we close the output channel
		defer close(out)

		// check whether an entry exists in the user-provided map, which MAY be nil
		// if/when the user has chosen their policy to be as such
		tactics, found := ldp.Root.DomainEndpoints[net.JoinHostPort(domain, port)]
		if !found {
			return
		}

		// make sure that there are actionable entries here
		tactics = userPolicyRemoveNilEntries(tactics)
		if len(tactics) <= 0 {
			return
		}

		// emit all the user-configured tactics
		for _, tactic := range tactics {
			out <- tactic
		}
	}()

	return out
}

// userPolicyKey is the kvstore key used to retrieve the user policy.
const userPolicyKey = "bridges.conf"

// errUserPolicyWrongVersion means that the user policy document has the wrong version number.
var errUserPolicyWrongVersion = errors.New("wrong user policy version")

// userPolicyVersion is the current version of the user policy file.
const userPolicyVersion = 3

// userPolicyRoot is the root of the user policy.
type userPolicyRoot struct {
	// DomainEndpoints maps each domain endpoint to its policies.
	DomainEndpoints map[string][]*httpsDialerTactic

	// Version is the data structure version.
	Version int
}

func userPolicyRemoveNilEntries(input []*httpsDialerTactic) (output []*httpsDialerTactic) {
	for _, entry := range input {
		if entry != nil {
			output = append(output, entry)
		}
	}
	return
}
