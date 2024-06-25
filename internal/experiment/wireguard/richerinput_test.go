package wireguard

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL: "wg://unknown.corp",
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
		if target.Input() != "wg://unknown.corp" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "wg://unknown.corp" {
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

	// make sure the loader is okay
	if child != loader.loader {
		t.Fatal("invalid loader pointer")
	}

	// make sure the options are okay
	if options != loader.options {
		t.Fatal("invalid options pointer")
	}
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
			name: "with options and inputs",
			options: &Config{
				SafeRemote: "1.1.1.1:443",
			},
			loader: &targetloading.Loader{
				ExperimentName: "wireguard",
				InputPolicy:    model.InputNone,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"wg://unknown.corp/1.1.1.1",
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL: "wg://unknown.corp/1.1.1.1",
					Options: &Config{
						SafeRemote: "1.1.1.1:443",
					},
				},
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create a target loader using the given config
			tl := &targetLoader{
				loader:  tc.loader,
				options: tc.options,
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
