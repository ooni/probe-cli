package testenv

import (
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func Test_haveIPv6(t *testing.T) {
	// testcase is a test case for this test
	type testcase struct {
		// name is the test case name
		name string

		// lookup is the DNS lookup function to pass to [haveIPv6]
		lookup func(domain string) ([]string, error)

		// dialTimeout is the dial function to pass to [haveIPv6]
		dialTimeout func(network string, address string, timeout time.Duration) (net.Conn, error)

		// expectResult is the expected result
		expectResult bool
	}

	// testcases contains all the test cases
	testcases := []testcase{{
		name: "when the DNS lookup fails",
		lookup: func(domain string) ([]string, error) {
			return nil, errors.New("mocked error")
		},
		dialTimeout:  nil,
		expectResult: false,
	}, {
		name: "when the lookup does not return any IPv6 entry",
		lookup: func(domain string) ([]string, error) {
			addrs := []string{
				"127.0.0.1",
			}
			return addrs, nil
		},
		dialTimeout:  nil,
		expectResult: false,
	}, {
		name: "when the lookup returns IPv6 entries and they don't work",
		lookup: func(domain string) ([]string, error) {
			addrs := []string{
				"127.0.0.1",
				"::1",
			}
			return addrs, nil
		},
		dialTimeout: func(network, address string, timeout time.Duration) (net.Conn, error) {
			return nil, errors.New("mocked error")
		},
		expectResult: false,
	}, {
		name: "when the lookup returns IPv6 entries and they work",
		lookup: func(domain string) ([]string, error) {
			addrs := []string{
				"127.0.0.1",
				"::1",
			}
			return addrs, nil
		},
		dialTimeout: func(network string, address string, timeout time.Duration) (net.Conn, error) {
			conn := &mocks.Conn{
				MockClose: func() error {
					return nil
				},
			}
			return conn, nil
		},
		expectResult: true,
	}}

	// run all the test cases
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			result := haveIPv6(tc.lookup, tc.dialTimeout)
			if tc.expectResult != result {
				t.Fatal("expected", tc.expectResult, "got", result)
			}
		})
	}
}
