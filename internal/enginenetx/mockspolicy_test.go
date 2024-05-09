package enginenetx

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

// mocksPolicy is a mockable policy
type mocksPolicy struct {
	MockLookupTactics func(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic
}

var _ httpsDialerPolicy = &mocksPolicy{}

// LookupTactics implements httpsDialerPolicy.
func (p *mocksPolicy) LookupTactics(ctx context.Context, domain string, port string) <-chan *httpsDialerTactic {
	return p.MockLookupTactics(ctx, domain, port)
}

func TestMocksPolicy(t *testing.T) {
	// create and fake fill a mocked policy
	var tx httpsDialerTactic
	ff := &testingx.FakeFiller{}
	ff.Fill(&tx)

	// create a mocks policy
	p := &mocksPolicy{
		MockLookupTactics: func(ctx context.Context, domain, port string) <-chan *httpsDialerTactic {
			output := make(chan *httpsDialerTactic, 1)
			output <- &tx
			close(output)
			return output
		},
	}

	// read the tactics emitted by the policy
	var got []*httpsDialerTactic
	for entry := range p.LookupTactics(context.Background(), "api.ooni.io", "443") {
		got = append(got, entry)
	}

	// make sure we've got what we expect
	if diff := cmp.Diff([]*httpsDialerTactic{&tx}, got); diff != "" {
		t.Fatal(diff)
	}
}
