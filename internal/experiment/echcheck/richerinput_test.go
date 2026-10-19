package echcheck

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL: "https://cloudflare-ech.com/cdn-cgi/trace",
		Config: &Config{
			ResolverURL: "https://mozilla.cloudflare-dns.com/dns-query",
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
		if target.Input() != "https://cloudflare-ech.com/cdn-cgi/trace" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		expect := []string{"ResolverURL=https://mozilla.cloudflare-dns.com/dns-query"}
		if diff := cmp.Diff(expect, target.Options()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "https://cloudflare-ech.com/cdn-cgi/trace" {
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
			options: &Config{ResolverURL: "https://dns.google/dns-query"},
			loader: &targetloading.Loader{
				ExperimentName: "echcheck",
				InputPolicy:    model.InputOptional,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs:   []string{"https://cloudflare-ech.com/cdn-cgi/trace"},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "https://cloudflare-ech.com/cdn-cgi/trace",
					Config: &Config{ResolverURL: "https://dns.google/dns-query"},
				},
			},
		},

		{
			name:    "per-input inputs_extra overlays the experiment-wide config",
			options: &Config{ResolverURL: "https://dns.google/dns-query"},
			loader: &targetloading.Loader{
				ExperimentName: "echcheck",
				InputPolicy:    model.InputOptional,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
				StaticInputs: []string{
					"https://cloudflare-ech.com/cdn-cgi/trace",
					"https://crypto.cloudflare.com/cdn-cgi/trace",
				},
				StaticInputsConfig: []json.RawMessage{
					json.RawMessage(`{"resolver_url":"https://1.1.1.1/dns-query"}`),
					json.RawMessage(`{}`),
				},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "https://cloudflare-ech.com/cdn-cgi/trace",
					Config: &Config{ResolverURL: "https://1.1.1.1/dns-query"}, // overridden per-input
				},
				&Target{
					URL:    "https://crypto.cloudflare.com/cdn-cgi/trace",
					Config: &Config{ResolverURL: "https://dns.google/dns-query"}, // unchanged
				},
			},
		},

		{
			name:    "no input returns a single empty-URL target because input is optional",
			options: &Config{ResolverURL: "https://dns.google/dns-query"},
			loader: &targetloading.Loader{
				ExperimentName: "echcheck",
				InputPolicy:    model.InputOptional,
				Logger:         model.DiscardLogger,
				Session:        &mocks.Session{},
			},
			expectErr: nil,
			expectTargets: []model.ExperimentTarget{
				&Target{
					URL:    "",
					Config: &Config{ResolverURL: "https://dns.google/dns-query"},
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
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}
			if diff := cmp.Diff(tc.expectTargets, targets); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
