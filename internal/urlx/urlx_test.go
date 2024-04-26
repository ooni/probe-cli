package urlx

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestResolveReference(t *testing.T) {
	// testcase is a test case used by this function
	type testcase struct {
		// Name is the test case name.
		Name string

		// BaseURL is the base URL.
		BaseURL string

		// Path is the extra path to append.
		Path string

		// RawQuery is the raw query.
		RawQuery string

		// ExpectErr is the expected err.
		ExpectErr error

		// ExpectURL is what we expect to see.
		ExpectURL string
	}

	cases := []testcase{{
		Name:      "when we cannot parse the base URL",
		BaseURL:   "\t", // invalid
		Path:      "",
		RawQuery:  "",
		ExpectErr: errors.New(`parse "\t": net/url: invalid control character in URL`),
		ExpectURL: "",
	}, {
		Name:      "when there's only the base URL",
		BaseURL:   "https://api.ooni.io/",
		Path:      "",
		RawQuery:  "",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/",
	}, {
		Name:      "when there's only the path",
		BaseURL:   "",
		Path:      "/api/v1/check-in",
		RawQuery:  "",
		ExpectErr: nil,
		ExpectURL: "/api/v1/check-in",
	}, {
		Name:      "when there's only the query",
		BaseURL:   "",
		Path:      "",
		RawQuery:  "key1=value1&key1=value2&key3=value3",
		ExpectErr: nil,
		ExpectURL: "?key1=value1&key1=value2&key3=value3",
	}, {
		Name:      "with base URL and path",
		BaseURL:   "https://api.ooni.io/",
		Path:      "/api/v1/check-in",
		RawQuery:  "",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/api/v1/check-in",
	}, {
		Name:      "with base URL and query",
		BaseURL:   "https://api.ooni.io/",
		Path:      "",
		RawQuery:  "key1=value1&key1=value2&key3=value3",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/?key1=value1&key1=value2&key3=value3",
	}, {
		Name:      "with base URL, path, and query",
		BaseURL:   "https://api.ooni.io/",
		Path:      "/api/v1/check-in",
		RawQuery:  "key1=value1&key1=value2&key3=value3",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/api/v1/check-in?key1=value1&key1=value2&key3=value3",
	}, {
		Name:      "with base URL with path, path, and query",
		BaseURL:   "https://api.ooni.io/api",
		Path:      "/v1/check-in",
		RawQuery:  "key1=value1&key1=value2&key3=value3",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/v1/check-in?key1=value1&key1=value2&key3=value3",
	}, {
		Name:      "with base URL with path and query, path, and query",
		BaseURL:   "https://api.ooni.io/api?foo=bar",
		Path:      "/v1/check-in",
		RawQuery:  "key1=value1&key1=value2&key3=value3",
		ExpectErr: nil,
		ExpectURL: "https://api.ooni.io/v1/check-in?key1=value1&key1=value2&key3=value3",
	}}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			// invoke the API we're currently testing
			got, err := ResolveReference(tc.BaseURL, tc.Path, tc.RawQuery)

			// check for errors
			switch {
			case err == nil && tc.ExpectErr == nil:
				if diff := cmp.Diff(tc.ExpectURL, got); diff != "" {
					t.Fatal(diff)
				}
				return

			case err != nil && tc.ExpectErr == nil:
				t.Fatal("expected err", tc.ExpectErr, "got", err)
				return

			case err == nil && tc.ExpectErr != nil:
				t.Fatal("expected err", tc.ExpectErr, "got", err)
				return

			case err != nil && tc.ExpectErr != nil:
				if err.Error() != tc.ExpectErr.Error() {
					t.Fatal("expected err", tc.ExpectErr, "got", err)
				}
				return
			}

		})
	}
}
