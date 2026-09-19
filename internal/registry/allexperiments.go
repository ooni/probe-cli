package registry

import (
	"sort"

	"github.com/ooni/probe-cli/v3/internal/experimentname"
)

// defaultVersion is the map key under which an experiment's default
// (unversioned) factory is registered.
const defaultVersion = ""

// experimentFactories contains the registered factories for the versions of a
// single experiment. The default factory is stored under [defaultVersion] and
// is what a bare name resolves to.
type experimentFactories struct {
	// factories maps a version key to the function constructing the
	// corresponding [*Factory]. The [defaultVersion] key holds the default.
	factories map[string]func() *Factory
}

// AllExperiments maps each experiment's base name to its registered versions.
var AllExperiments = map[string]*experimentFactories{}

// register registers the default (unversioned) factory for the given base name.
func register(base string, f func() *Factory) {
	registerVersion(base, defaultVersion, f)
}

// registerVersion registers a factory for a specific version of an experiment
// identified by its base name. The [defaultVersion] key is the default.
func registerVersion(base, version string, f func() *Factory) {
	entry := AllExperiments[base]
	if entry == nil {
		entry = &experimentFactories{
			factories: map[string]func() *Factory{},
		}
		AllExperiments[base] = entry
	}
	// Wrap the factory so the canonical name is always the base name,
	// regardless of the selected version.
	entry.factories[version] = func() *Factory {
		factory := f()
		factory.canonicalName = base
		return factory
	}
}

// ExperimentNames returns the full canonical names of all registered
// experiments. The output is sorted.
func ExperimentNames() (names []string) {
	for base, entry := range AllExperiments {
		for version := range entry.factories {
			names = append(names, experimentname.Name{Base: base, Version: version}.String())
		}
	}
	sort.Strings(names) // sort by name to always provide predictable output
	return
}

// RegisteredFactories returns a map from each experiment's full canonical name
// (base, or "base@version") to the function constructing its factory.
func RegisteredFactories() map[string]func() *Factory {
	out := map[string]func() *Factory{}
	for base, entry := range AllExperiments {
		for version, f := range entry.factories {
			out[experimentname.Name{Base: base, Version: version}.String()] = f
		}
	}
	return out
}
