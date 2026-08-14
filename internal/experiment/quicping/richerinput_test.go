package quicping

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL: "8.8.8.8",
		Config: &Config{
			Port:        443,
			Repetitions: 4,
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
		if target.Input() != "8.8.8.8" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		expect := []string{"Repetitions=4", "Port=443"}
		if diff := cmp.Diff(expect, target.Options()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "8.8.8.8" {
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
			name:    "with options and inputs",
			options: &Config{Repetitions: 3},
			loader: &targetloading.Loader{
				ExperimentName: "quicping",
				InputPolicy:    model.InputStrictlyRequired,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"1.1.1.1"},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "1.1.1.1",
					Config: &Config{Repetitions: 3},
				},
			},
		},

		{
			name:    "per-input inputs_extra overlays the experiment-wide config",
			options: &Config{Repetitions: 3},
			loader: &targetloading.Loader{
				ExperimentName: "quicping",
				InputPolicy:    model.InputStrictlyRequired,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"1.1.1.1",
					"8.8.8.8",
				},
				StaticInputsConfig: []json.RawMessage{
					json.RawMessage(`{"port":8443}`),
					json.RawMessage(`{}`),
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "1.1.1.1",
					Config: &Config{Repetitions: 3, Port: 8443}, // overridden per-input
				},
				&Target{
					URL:    "8.8.8.8",
					Config: &Config{Repetitions: 3}, // unchanged
				},
			},
		},

		{
			name:    "no input errors because input is strictly required",
			options: &Config{},
			loader: &targetloading.Loader{
				ExperimentName: "quicping",
				InputPolicy:    model.InputStrictlyRequired,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
			},
			expectErr:     targetloading.ErrInputRequired,
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
			// Config carries an unexported func field (netListenUDP) that cmp
			// cannot compare, so ignore unexported fields here.
			if diff := cmp.Diff(tc.expectTargets, targets, cmpopts.IgnoreUnexported(Config{})); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
