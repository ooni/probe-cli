package telegram

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
		URL:    "",
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
		if target.Input() != "" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		// The telegram Config carries no options.
		if len(target.Options()) != 0 {
			t.Fatal("expected no options")
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "" {
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
			name:    "no input returns a single config-only target because input is not expected",
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "telegram",
				InputPolicy:    model.InputNone,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{URL: "", Config: &Config{}},
			},
		},

		{
			name:    "with static input we return ErrNoInputExpected",
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "telegram",
				InputPolicy:    model.InputNone,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"https://www.example.com/"},
			},
			expectErr:     targetloading.ErrNoInputExpected,
			expectTargets: nil,
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
