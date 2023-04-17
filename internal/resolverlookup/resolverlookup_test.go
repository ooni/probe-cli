package resolverlookup

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testenv"
)

func TestClient(t *testing.T) {
	// testcase is a test case for this test
	type testcase struct {
		// name is the test case name
		name string

		// needsIPv6 indicates whether this test case requires IPv6
		needsIPv6 bool

		// newContext is the factory to create a new context
		newContext func() (context.Context, context.CancelFunc)

		// fx is the function to execute
		fx func(context.Context) (string, error)

		// expectedErr is the expected error
		expectedErr error

		// expectNonEmptyAddr is the expected addr
		expectNonEmptyAddr bool
	}

	// client is the client we'll use in this test
	client := &Client{
		Logger: model.DiscardLogger,
	}

	// testcases contains all test cases
	testcases := []testcase{{
		name:      "successful IPv4 lookup",
		needsIPv6: false,
		newContext: func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		},
		fx:                 client.LookupResolverIPv4,
		expectedErr:        nil,
		expectNonEmptyAddr: true,
	}, {
		name:      "failing IPv4 lookup",
		needsIPv6: false,
		newContext: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // make sure we fail immediately
			return ctx, cancel
		},
		fx:                 client.LookupResolverIPv4,
		expectedErr:        context.Canceled,
		expectNonEmptyAddr: false,
	}, {
		name:      "successful IPv6 lookup",
		needsIPv6: true,
		newContext: func() (context.Context, context.CancelFunc) {
			return context.WithCancel(context.Background())
		},
		fx:                 client.LookupResolverIPv6,
		expectedErr:        nil,
		expectNonEmptyAddr: true,
	}, {
		name:      "failing IPv6 lookup",
		needsIPv6: false,
		newContext: func() (context.Context, context.CancelFunc) {
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // make sure we fail immediately
			return ctx, cancel
		},
		fx:                 client.LookupResolverIPv6,
		expectedErr:        context.Canceled,
		expectNonEmptyAddr: false,
	}}

	// run all the test cases
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// avoid running this test if we don't have IPv6 support
			if tc.needsIPv6 && !testenv.SupportsIPv6 {
				t.Skip("skip test requiring IPv6")
			}

			// create the context
			ctx, cancel := tc.newContext()
			defer cancel()

			// call the function to lookup the resolver
			addr, err := tc.fx(ctx)

			// make sure the error is the expected one
			if !errors.Is(err, tc.expectedErr) {
				t.Fatal("unexpected error", err)
			}

			// make sure the returned address is the expected one
			switch {
			case tc.expectNonEmptyAddr && addr == "":
				t.Fatal("expected a non-empty string")
			case !tc.expectNonEmptyAddr && addr != "":
				t.Fatal("expected an empty string")
			}
		})
	}
}
