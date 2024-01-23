package idnax

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestToASCII(t *testing.T) {
	// testcase is a test case implemented by this function.
	type testcase struct {
		// input is the input domain
		input string

		// expectErr is the expected error
		expectErr error

		// expectDomain is the expected domain
		expectDomain string
	}

	testcases := []testcase{{
		input:        "ουτοπία.δπθ.gr",
		expectErr:    nil,
		expectDomain: "xn--kxae4bafwg.xn--pxaix.gr",
	}, {
		input:        "example.com",
		expectErr:    nil,
		expectDomain: "example.com",
	}, {
		input:        "Яндекс.рф",
		expectErr:    nil,
		expectDomain: "xn--d1acpjx3f.xn--p1ai",
	}, {
		// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
		input:        "http://xn--0000h/",
		expectErr:    errors.New("idna: disallowed rune U+003A"),
		expectDomain: "",
	}}

	for _, tc := range testcases {
		t.Run(tc.input, func(t *testing.T) {
			output, err := ToASCII(tc.input)

			switch {
			case err == nil && tc.expectErr == nil:
				if diff := cmp.Diff(tc.expectDomain, output); diff != "" {
					t.Fatal(diff)
				}

			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)

			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)

			case err != nil && tc.expectErr != nil:
				if err.Error() != tc.expectErr.Error() {
					t.Fatal("expected", tc.expectErr, "got", err)
				}
			}
		})
	}
}
