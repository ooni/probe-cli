package inputparser

import (
	"errors"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestParse(t *testing.T) {

	// testCase describes a test case.
	type testCase struct {
		// name is the MANDATORY name of the test case.
		name string

		// config is the MANDATORY Config to use.
		config *Config

		// input is the MANDATORY string-format input-URL to parse.
		input model.MeasurementTarget

		// expectURL is the OPTIONAL URL we expect in output.
		expectURL *url.URL

		// expectErr is the OPTIONAL error we expect in output.
		expectErr error
	}

	var allTestCases = []testCase{{
		name: "when the input is an endpoint and we accept endpoints",
		config: &Config{
			// We don't need to provide an AcceptedScheme when ONLY parsing endpoints.
			AcceptedSchemes: []string{""},
			AllowEndpoints:  true,
			DefaultScheme:   "http",
		},
		input: "example.com:80",
		expectURL: &url.URL{
			Scheme: "http",
			Host:   "example.com:80",
		},
		expectErr: nil,
	}, {
		name: "when the input is an endpoint and we don't accept endpoints",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		input:     "example.com:80",
		expectURL: nil,
		expectErr: ErrEmptyHostname,
	}, {
		name: "when the input is a domain or IP address and we accept endpoints",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  true,
			DefaultScheme:   "http",
		},
		input:     "example.com",
		expectURL: nil,
		expectErr: ErrInvalidEndpoint,
	}, {
		name: "when the URL does not parse",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		input:     "http://\t/\r\n",
		expectURL: nil,
		expectErr: ErrURLParse,
	}, {
		name: "when the URL scheme is unsupported",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		input:     "smtp://example.com:53",
		expectURL: nil,
		expectErr: ErrUnsupportedScheme,
	}, {
		name: "when the default scheme is empty",
		config: &Config{
			AcceptedSchemes: []string{},
			AllowEndpoints:  true,
			DefaultScheme:   "",
		},
		input:     "example.com:80",
		expectURL: nil,
		expectErr: ErrEmptyDefaultScheme,
	}, {
		name: "for IDNA URL without a port",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		input: "http://ουτοπία.δπθ.gr/",
		expectURL: &url.URL{
			Scheme: "http",
			Host:   "xn--kxae4bafwg.xn--pxaix.gr",
			Path:   "/",
		},
		expectErr: nil,
	}, {
		name: "for IDNA URL with a port",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		input: "http://ουτοπία.δπθ.gr:80/",
		expectURL: &url.URL{
			Scheme: "http",
			Host:   "xn--kxae4bafwg.xn--pxaix.gr:80",
			Path:   "/",
		},
		expectErr: nil,
	}, {
		name: "when we cannot convert IDNA to ASCII",
		config: &Config{
			AcceptedSchemes: []string{"http"},
			AllowEndpoints:  false,
			DefaultScheme:   "",
		},
		// See https://www.farsightsecurity.com/blog/txt-record/punycode-20180711/
		input:     "http://xn--0000h/",
		expectURL: nil,
		expectErr: ErrIDNAToASCII,
	}}

	for _, tc := range allTestCases {
		t.Run(tc.name, func(t *testing.T) {
			URL, err := Parse(tc.config, tc.input)

			// parse the error
			switch {
			case err == nil && tc.expectErr == nil:
				// nothing
			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)
			default:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
			}

			// validate the returned URL
			if diff := cmp.Diff(tc.expectURL, URL); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
