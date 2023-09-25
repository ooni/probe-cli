package enginenetx

import (
	"context"
	"errors"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/hujsonx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPSDialerStaticPolicy is an [HTTPSDialerPolicy] incorporating verbatim
// a static policy loaded from the engine's key-value store.
//
// This policy is very useful for exploration and experimentation.
type HTTPSDialerStaticPolicy struct {
	// Fallback is the fallback policy in case the static one does not
	// contain a rule for a specific domain.
	Fallback HTTPSDialerPolicy

	// Root is the root of the statically loaded policy.
	Root *HTTPSDialerStaticPolicyRoot
}

// HTTPSDialerStaticPolicyKey is the kvstore key used to retrieve the static policy.
const HTTPSDialerStaticPolicyKey = "httpsdialerstatic.conf"

// errDialerStaticPolicyWrongVersion means that the static policy document has the wrong version number.
var errDialerStaticPolicyWrongVersion = errors.New("wrong static policy version")

// NewHTTPSDialerStaticPolicy attempts to constructs a static policy using a given fallback
// policy and either returns a good policy or an error. The typical error case is the one
// in which there's no httpsDialerStaticPolicyKey in the key-value store.
func NewHTTPSDialerStaticPolicy(
	kvStore model.KeyValueStore, fallback HTTPSDialerPolicy) (*HTTPSDialerStaticPolicy, error) {
	// attempt to read the static policy bytes from the kvstore
	data, err := kvStore.Get(HTTPSDialerStaticPolicyKey)
	if err != nil {
		return nil, err
	}

	// attempt to parse the static policy using human-readable JSON
	var root HTTPSDialerStaticPolicyRoot
	if err := hujsonx.Unmarshal(data, &root); err != nil {
		return nil, err
	}

	// make sure the version is OK
	if root.Version != HTTPSDialerStaticPolicyVersion {
		err := fmt.Errorf(
			"%s: %w: expected=%d got=%d",
			HTTPSDialerStaticPolicyKey,
			errDialerStaticPolicyWrongVersion,
			HTTPSDialerStaticPolicyVersion,
			root.Version,
		)
		return nil, err
	}

	out := &HTTPSDialerStaticPolicy{
		Fallback: fallback,
		Root:     &root,
	}
	return out, nil
}

// HTTPSDialerStaticPolicyVersion is the current version of the static policy file.
const HTTPSDialerStaticPolicyVersion = 1

// HTTPSDialerStaticPolicyRoot is the root of a statically loaded policy.
type HTTPSDialerStaticPolicyRoot struct {
	// Domains maps each domain to its policy.
	Domains map[string][]*HTTPSDialerTactic

	// Version is the data structure version.
	Version int
}

var _ HTTPSDialerPolicy = &HTTPSDialerStaticPolicy{}

// LookupTactics implements HTTPSDialerPolicy.
func (ldp *HTTPSDialerStaticPolicy) LookupTactics(
	ctx context.Context, domain string, port string) <-chan *HTTPSDialerTactic {
	tactics, found := ldp.Root.Domains[domain]
	if !found {
		return ldp.Fallback.LookupTactics(ctx, domain, port)
	}

	out := make(chan *HTTPSDialerTactic)
	go func() {
		defer close(out)
		for _, tactic := range tactics {
			out <- tactic
		}
	}()
	return out
}
