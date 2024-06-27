package dnscheck

import (
	"context"
	"errors"
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL: "https://dns.google/dns-query",
		Config: &Config{
			DefaultAddrs:  "8.8.8.8 8.8.4.4",
			Domain:        "example.com",
			HTTP3Enabled:  false,
			HTTPHost:      "dns.google",
			TLSServerName: "dns.google.com",
			TLSVersion:    "TLSv1.3",
		},
	}

	t.Run("Category", func(t *testing.T) {
		if target.Category() != model.DefaultCategoryCode {
			t.Fatal("invalid Category")
		}
	})

	t.Run("Country", func(t *testing.T) {
		if target.Country() != model.DefaultCountryCode {
			t.Fatal("invalid Country")
		}
	})

	t.Run("Input", func(t *testing.T) {
		if target.Input() != "https://dns.google/dns-query" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		expect := []string{
			"DefaultAddrs=8.8.8.8 8.8.4.4",
			"Domain=example.com",
			"HTTPHost=dns.google",
			"TLSServerName=dns.google.com",
			"TLSVersion=TLSv1.3",
		}
		if diff := cmp.Diff(expect, target.Options()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "https://dns.google/dns-query" {
			t.Fatal("invalid String")
		}
	})
}

func TestNewLoader(t *testing.T) {
	// create the pointers we expect to see
	child := &targetloading.Loader{}
	options := &Config{}

	// create the loader and cast it to its private type
	loader := NewLoader(child, options).(*targetLoader)

	// make sure the default input is okay
	if diff := cmp.Diff(defaultInput, loader.defaultInput); diff != "" {
		t.Fatal(diff)
	}

	// make sure the loader is okay
	if child != loader.loader {
		t.Fatal("invalid loader pointer")
	}

	// make sure the options are okay
	if options != loader.options {
		t.Fatal("invalid options pointer")
	}
}

// testDefaultInput is the default input used by [TestTargetLoaderLoad].
var testDefaultInput = []model.ExperimentTarget{
	&Target{
		URL: "https://dns.google/dns-query",
		Config: &Config{
			HTTP3Enabled: true,
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},
	&Target{
		URL: "https://dns.google/dns-query",
		Config: &Config{
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},
}

func TestTargetLoaderLoad(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// options contains the options to use
		options *Config

		// loader is the loader to use
		loader *targetloading.Loader

		// expectErr is the error we expect
		expectErr error

		// expectResults contains the expected results
		expectTargets []model.ExperimentTarget
	}

	cases := []testcase{

		{
			name: "with options, inputs, and files",
			options: &Config{
				DefaultAddrs: "1.1.1.1 1.0.0.1",
			},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"https://dns.cloudflare.com/dns-query",
					"https://one.one.one.one/dns-query",
				},
				SourceFiles: []string{
					filepath.Join("testdata", "input.txt"),
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL: "https://dns.cloudflare.com/dns-query",
					Config: &Config{
						DefaultAddrs: "1.1.1.1 1.0.0.1",
					},
				},
				&Target{
					URL: "https://one.one.one.one/dns-query",
					Config: &Config{
						DefaultAddrs: "1.1.1.1 1.0.0.1",
					},
				},
				&Target{
					URL: "https://1dot1dot1dot1dot.com/dns-query",
					Config: &Config{
						DefaultAddrs: "1.1.1.1 1.0.0.1",
					},
				},
				&Target{
					URL: "https://dns.cloudflare/dns-query",
					Config: &Config{
						DefaultAddrs: "1.1.1.1 1.0.0.1",
					},
				},
			},
		},

		{
			name: "with an unreadable file",
			options: &Config{
				DefaultAddrs: "1.1.1.1 1.0.0.1",
			},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"https://dns.cloudflare.com/dns-query",
					"https://one.one.one.one/dns-query",
				},
				SourceFiles: []string{
					filepath.Join("testdata", "nonexistent.txt"),
				},
			},
			expectErr:     fs.ErrNotExist,
			expectTargets: nil,
		},

		{
			name:    "with just inputs",
			options: &Config{},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"https://dns.cloudflare.com/dns-query",
					"https://one.one.one.one/dns-query",
				},
				SourceFiles: []string{},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "https://dns.cloudflare.com/dns-query",
					Config: &Config{},
				},
				&Target{
					URL:    "https://one.one.one.one/dns-query",
					Config: &Config{},
				},
			},
		},

		{
			name:    "with just files",
			options: &Config{},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{},
				SourceFiles: []string{
					filepath.Join("testdata", "input.txt"),
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "https://1dot1dot1dot1dot.com/dns-query",
					Config: &Config{},
				},
				&Target{
					URL:    "https://dns.cloudflare/dns-query",
					Config: &Config{},
				},
			},
		},

		{
			name: "with just options",
			options: &Config{
				DefaultAddrs: "1.1.1.1 1.0.0.1",
			},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{},
				SourceFiles:    []string{},
			},
			expectErr:     nil,
			expectTargets: nil,
		},

		{
			name:    "with no options, not inputs, no files",
			options: &Config{},
			loader: &targetloading.Loader{
				CheckInConfig:  &model.OOAPICheckInConfig{ /* nothing */ },
				ExperimentName: "dnscheck",
				InputPolicy:    model.InputOrStaticDefault,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{},
				SourceFiles:    []string{},
			},
			expectErr:     nil,
			expectTargets: testDefaultInput,
		}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create a target loader using the given config
			//
			// note that we use a default test input for results predictability
			// since the static list may change over time
			tl := &targetLoader{
				defaultInput: testDefaultInput,
				loader:       tc.loader,
				options:      tc.options,
			}

			// load targets
			targets, err := tl.Load(context.Background())

			// make sure error is consistent
			switch {
			case err == nil && tc.expectErr == nil:
				// fallthrough

			case err != nil && tc.expectErr != nil:
				if !errors.Is(err, tc.expectErr) {
					t.Fatal("unexpected error", err)
				}
				// fallthrough

			default:
				t.Fatal("expected", tc.expectErr, "got", err)
			}

			// make sure the targets are consistent
			if diff := cmp.Diff(tc.expectTargets, targets); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
