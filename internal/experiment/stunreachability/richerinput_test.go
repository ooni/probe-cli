package stunreachability

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func TestTarget(t *testing.T) {
	target := &Target{
		URL:    "stun://stun.ekiga.net:3478",
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
		if target.Input() != "stun://stun.ekiga.net:3478" {
			t.Fatal("invalid Input")
		}
	})

	t.Run("Options", func(t *testing.T) {
		// The stunreachability Config carries only unexported test seams.
		if len(target.Options()) != 0 {
			t.Fatal("expected no options")
		}
	})

	t.Run("String", func(t *testing.T) {
		if target.String() != "stun://stun.ekiga.net:3478" {
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

func TestTargetLoaderLoadWithInputs(t *testing.T) {
	tl := &targetLoader{
		options: &Config{},
		loader: &targetloading.Loader{
			ExperimentName: "stunreachability",
			InputPolicy:    model.InputOrStaticDefault,
			Logger:         model.DiscardLogger,
			Session:        &mocks.Session{},
			StaticInputs:   []string{"stun://stun.ekiga.net:3478"},
		},
	}
	targets, err := tl.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	expect := []model.ExperimentTarget{
		&Target{URL: "stun://stun.ekiga.net:3478", Config: &Config{}},
	}
	if diff := cmp.Diff(expect, targets, cmpopts.IgnoreUnexported(Config{})); diff != "" {
		t.Fatal(diff)
	}
}

func TestTargetLoaderLoadFallsBackToStaticDefault(t *testing.T) {
	tl := &targetLoader{
		defaultInput: defaultInputTargets,
		options:      &Config{},
		loader: &targetloading.Loader{
			ExperimentName: "stunreachability",
			InputPolicy:    model.InputOrStaticDefault,
			Logger:         model.DiscardLogger,
			Session:        &mocks.Session{},
		},
	}
	targets, err := tl.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// With no input, the loader returns the built-in STUN endpoint list. Note
	// that stuninput.AsnStunReachabilityInput iterates a map, so its order is
	// randomized per call: compare as an unordered set of URLs.
	expectedURLs := stuninput.AsnStunReachabilityInput()
	if len(targets) != len(expectedURLs) {
		t.Fatalf("unexpected number of targets: got %d, want %d", len(targets), len(expectedURLs))
	}
	want := make(map[string]bool)
	for _, url := range expectedURLs {
		want[url] = true
	}
	for i, target := range targets {
		concrete, ok := target.(*Target)
		if !ok {
			t.Fatalf("target %d is not a *Target", i)
		}
		if !want[concrete.URL] {
			t.Fatalf("target %d: unexpected URL %q", i, concrete.URL)
		}
	}
}
