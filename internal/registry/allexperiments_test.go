package registry

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestRegisterExperimentPanicsWhenDefaultNotRegistered(t *testing.T) {
	const fakeBase = "this_is_not_a_real_experiment"
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected registerExperiment to panic, got no panic")
		}
		// the panic must happen before we mutate the global registry
		if _, ok := AllExperiments[fakeBase]; ok {
			t.Fatal("the registry must not be mutated when registration panics")
		}
	}()
	registerExperiment(fakeBase, "v2.0", map[string]func() *Factory{
		"v1.0": func() *Factory { return &Factory{} },
	})
}

func TestExperimentNamesDisplayDefaultsAsBare(t *testing.T) {
	names := RegisteredFactories()

	mustContain := func(name string) {
		if _, ok := names[name]; !ok {
			t.Fatalf("expected RegisteredFactories to contain %q", name)
		}
	}
	mustNotContain := func(name string) {
		if _, ok := names[name]; ok {
			t.Fatalf("expected RegisteredFactories NOT to contain %q", name)
		}
	}

	// web_connectivity's default is v0.4 (bare); v0.5 is the non-default.
	mustContain("web_connectivity")
	mustContain("web_connectivity@v0.5")
	mustNotContain("web_connectivity@v0.4") // the default must never appear with its @version

	// a single-version experiment appears only under its bare name.
	mustContain("urlgetter")
	mustNotContain("urlgetter@v0.2")

	// ExperimentNames must agree with RegisteredFactories.
	namesSet := map[string]bool{}
	for _, name := range ExperimentNames() {
		namesSet[name] = true
	}
	for name := range names {
		if !namesSet[name] {
			t.Fatalf("ExperimentNames is missing %q present in RegisteredFactories", name)
		}
	}
}

func TestNewFactoryResolvesConcreteDefaultVersion(t *testing.T) {
	store := &kvstore.Memory{}

	// a bare name resolves to the declared default version
	if _, err := NewFactory("urlgetter", store, model.DiscardLogger); err != nil {
		t.Fatalf("bare name failed to resolve: %v", err)
	}

	// the explicit default-version selector resolves too
	if _, err := NewFactory("urlgetter@v0.2", store, model.DiscardLogger); err != nil {
		t.Fatalf("explicit default version failed to resolve: %v", err)
	}

	// an unknown version is an error
	if _, err := NewFactory("urlgetter@v9.9", store, model.DiscardLogger); !errors.Is(err, ErrNoSuchExperiment) {
		t.Fatalf("expected ErrNoSuchExperiment for unknown version, got %v", err)
	}
}
