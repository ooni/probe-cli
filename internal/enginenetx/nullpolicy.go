package enginenetx

//
// A policy that never returns any tactic.
//

import "context"

// nullPolicy is a policy that never returns any tactics.
//
// You can use this policy to terminate the policy chain and
// ensure ane existing policy has a "null" fallback.
//
// The zero value is ready to use.
type nullPolicy struct{}

var _ httpsDialerPolicy = &nullPolicy{}

// LookupTactics implements httpsDialerPolicy.
//
// This policy returns a closed channel such that it won't
// be possible to read policies from it.
func (n *nullPolicy) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	output := make(chan *httpsDialerTactic)
	close(output)
	return output
}
