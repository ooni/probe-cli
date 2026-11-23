package registry

import (
	"fmt"
	"sort"

	"github.com/ooni/probe-cli/v3/internal/experimentname"
)

// experimentFactories contains the registered factories for the versions of a
// single experiment. Each experiment declares a concrete [defaultVersion] that a
// bare name resolves to.
type experimentFactories struct {
	// defaultVersion is the concrete version key that a bare name resolves to.
	// It is always one of the keys in [factories].
	defaultVersion string

	// factories maps a concrete version key to the function constructing the
	// corresponding [*Factory].
	factories map[string]func() *Factory
}

// AllExperiments maps each experiment's base name to its registered versions.
var AllExperiments = map[string]*experimentFactories{}

// registerExperiment registers one or more versioned factories for the given
// base name and declares which version is the default that a bare name resolves
// to.
func registerExperiment(base, defaultVersion string, versions map[string]func() *Factory) {
	if _, ok := versions[defaultVersion]; !ok {
		panic(fmt.Sprintf("registry: default version %q not registered for %q", defaultVersion, base))
	}
	entry := AllExperiments[base]
	if entry == nil {
		entry = &experimentFactories{factories: map[string]func() *Factory{}}
		AllExperiments[base] = entry
	}
	entry.defaultVersion = defaultVersion
	for version, f := range versions {
		// Wrap the factory so the canonical name is always the base name,
		// regardless of the selected version.
		entry.factories[version] = func() *Factory {
			factory := f()
			factory.canonicalName = base
			return factory
		}
	}
}

// register registers a single-version experiment under the author-declared
// version, which also becomes its default.
func register(base, version string, f func() *Factory) {
	registerExperiment(base, version, map[string]func() *Factory{version: f})
}

// ExperimentNames returns the full canonical names of all registered
// experiments.
func ExperimentNames() (names []string) {
	for base, entry := range AllExperiments {
		for version := range entry.factories {
			names = append(names, displayName(base, version, entry.defaultVersion))
		}
	}
	sort.Strings(names) // sort by name to always provide predictable output
	return
}

// RegisteredFactories returns a map from each experiment's full canonical name
// to the function constructing its factory.
func RegisteredFactories() map[string]func() *Factory {
	out := map[string]func() *Factory{}
	for base, entry := range AllExperiments {
		for version, f := range entry.factories {
			out[displayName(base, version, entry.defaultVersion)] = f
		}
	}
	return out
}

// displayName returns the bare base name for the default version and the
// "base@version" canonical name otherwise.
func displayName(base, version, defaultVersion string) string {
	if version == defaultVersion {
		return base
	}
	return experimentname.Name{Base: base, Version: version}.String()
}
