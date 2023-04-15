package iplookup

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// This test ensures that we correctly handle errors in Client.LookupWithCloudflare.
func TestClientLookupWithCloudflare(t *testing.T) {
	// testcase is a test case in this test
	type testcase struct {
		// name is the test case name
		name string

		// fx is the function to initialize TestingHTTPDo
		fx func(req *http.Request) ([]byte, error)

		// expectErr is the expected error
		expectErr error

		// expectAddr is the expected IP addr
		expectedAddr string
	}

	// errMocked an error returned to pretend that something failed.
	errMocked := errors.New("mocked error")

	// testcases contains all the test cases.
	testcases := []testcase{{
		name: "httpDo fails",
		fx: func(req *http.Request) ([]byte, error) {
			return nil, errMocked
		},
		expectErr:    errMocked,
		expectedAddr: "",
	}, {
		name: "the response is empty",
		fx: func(req *http.Request) ([]byte, error) {
			return nil, nil
		},
		expectErr:    ErrInvalidIPAddress,
		expectedAddr: "",
	}, {
		name: "the response contains an invalid IP address",
		fx: func(req *http.Request) ([]byte, error) {
			response := []byte(`fl=270f97
			h=www.cloudflare.com
			ip=1.ehlo.4.7
			ts=1681469211.128
			visit_scheme=https
			uag=Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/111.0
			colo=MXP
			sliver=none
			http=http/3
			loc=IT
			tls=TLSv1.3
			sni=plaintext
			warp=off
			gateway=off
			rbi=off
			kex=X25519`)
			return response, nil
		},
		expectErr:    ErrInvalidIPAddress,
		expectedAddr: "",
	}, {
		name: "the response contains a valid IP address",
		fx: func(req *http.Request) ([]byte, error) {
			response := []byte(`fl=270f97
			h=www.cloudflare.com
			ip=1.4.4.7
			ts=1681469211.128
			visit_scheme=https
			uag=Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:109.0) Gecko/20100101 Firefox/111.0
			colo=MXP
			sliver=none
			http=http/3
			loc=IT
			tls=TLSv1.3
			sni=plaintext
			warp=off
			gateway=off
			rbi=off
			kex=X25519`)
			return response, nil
		},
		expectErr:    nil,
		expectedAddr: "1.4.4.7",
	}, {
		name: "we set a deadline for the request context",
		fx: func(req *http.Request) ([]byte, error) {
			ctx := req.Context()
			if _, ok := ctx.Deadline(); !ok {
				return nil, errors.New("missing deadline")
			}
			return []byte("ip=1.4.4.7"), nil
		},
		expectErr:    nil,
		expectedAddr: "1.4.4.7",
	}}

	// run each test case
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			// create a suitable client instance
			c := &Client{
				Logger:        model.DiscardLogger,
				Resolver:      netxlite.NewStdlibResolver(model.DiscardLogger),
				TestingHTTPDo: tc.fx,
			}

			// attempt to lookup
			addr, err := c.LookupIPAddr(context.Background(), MethodWebClouflare, FamilyINET)

			// make sure the error is the expected one
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}

			// make sure the address is the expected one
			if addr != tc.expectedAddr {
				t.Fatal("expected ", tc.expectErr, "got", addr)
			}
		})
	}
}
