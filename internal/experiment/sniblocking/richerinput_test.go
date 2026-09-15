package sniblocking

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL: "kernel.org",
		Config: &Config{
			ControlSNI: "example.org",
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
		if target.Input() != "kernel.org" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		expect := []string{"ControlSNI=example.org"}
		if diff := cmp.Diff(expect, target.Options()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "kernel.org" {
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
			name:    "with options and static inputs",
			options: &Config{ControlSNI: "example.org"},
			loader: &targetloading.Loader{
				ExperimentName: "sni_blocking",
				InputPolicy:    model.InputOrQueryBackend,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"kernel.org"},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "kernel.org",
					Config: &Config{ControlSNI: "example.org"},
				},
			},
		},

		{
			name:    "per-input inputs_extra overlays the experiment-wide config",
			options: &Config{ControlSNI: "example.org"},
			loader: &targetloading.Loader{
				ExperimentName: "sni_blocking",
				InputPolicy:    model.InputOrQueryBackend,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"kernel.org", "ooni.org"},
				StaticInputsConfig: []json.RawMessage{
					json.RawMessage(`{"control_sni":"example.com"}`),
					json.RawMessage(`{}`),
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "kernel.org",
					Config: &Config{ControlSNI: "example.com"}, // overridden per-input
				},
				&Target{
					URL:    "ooni.org",
					Config: &Config{ControlSNI: "example.org"}, // unchanged
				},
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
			if err != tc.expectErr {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(tc.expectTargets, targets); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
