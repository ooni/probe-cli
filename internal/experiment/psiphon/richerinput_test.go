package psiphon

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
		URL:    "https://www.example.com/",
		Config: &Config{},
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
		if target.Input() != "https://www.example.com/" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		// The psiphon Config carries no options yet.
		if len(target.Options()) != 0 {
			t.Fatal("expected no options")
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "https://www.example.com/" {
			t.Fatal("invalid String")
		}
	})
}

func TestNewLoader(t *testing.T) {
	child := &targetloading.Loader{}
	options := &Config{}
	loader := NewLoader(child, options).(*targetLoader)
	if child != loader.loader {
		t.Fatal("invalid loader pointer")
	}
	if options != loader.options {
		t.Fatal("invalid options pointer")
	}
}

func TestTargetLoaderLoad(t *testing.T) {
	type testcase struct {
		name          string
		options       *Config
		loader        *targetloading.Loader
		expectErr     error
		expectTargets []model.ExperimentTarget
	}

	cases := []testcase{
		{
			name:    "with inputs",
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "psiphon",
				InputPolicy:    model.InputOptional,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"https://www.example.com/",
					"https://www.example.org/",
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{URL: "https://www.example.com/", Config: &Config{}},
				&Target{URL: "https://www.example.org/", Config: &Config{}},
			},
		},

		{
			name:    "no input returns a single empty-URL target because input is optional",
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "psiphon",
				InputPolicy:    model.InputOptional,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{URL: "", Config: &Config{}},
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tl := &targetLoader{
				loader:  tc.loader,
				options: tc.options,
			}
			targets, err := tl.Load(context.Background())
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(tc.expectTargets, targets); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
