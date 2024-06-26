package dnscheck

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/experimentconfig"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/reflectx"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Target is a richer-input target that this experiment should measure.
type Target struct {
	// Config contains the configuration.
	Config *Config

	// URL is the input URL.
	URL string
}

var _ model.ExperimentTarget = &Target{}

// Category implements [model.ExperimentTarget].
func (t *Target) Category() string {
	return model.DefaultCategoryCode
}

// Country implements [model.ExperimentTarget].
func (t *Target) Country() string {
	return model.DefaultCountryCode
}

// Input implements [model.ExperimentTarget].
func (t *Target) Input() string {
	return t.URL
}

// Options implements [model.ExperimentTarget].
func (t *Target) Options() []string {
	return experimentconfig.DefaultOptionsSerializer(t.Config)
}

// String implements [model.ExperimentTarget].
func (t *Target) String() string {
	return t.URL
}

// NewLoader constructs a new [model.ExperimentTargerLoader] instance.
//
// This function PANICS if options is not an instance of [*dnscheck.Config].
func NewLoader(loader *targetloading.Loader, gopts any) model.ExperimentTargetLoader {
	// Panic if we cannot convert the options to the expected type.
	//
	// We do not expect a panic here because the type is managed by the registry package.
	options := gopts.(*Config)

	// Construct the proper loader instance.
	return &targetLoader{
		defaultInput: defaultInput,
		loader:       loader,
		options:      options,
	}
}

// targetLoader loads targets for this experiment.
type targetLoader struct {
	defaultInput []model.ExperimentTarget
	loader       *targetloading.Loader
	options      *Config
}

// Load implements model.ExperimentTargetLoader.
func (tl *targetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	// If inputs and files are all empty and there are no options, let's use the backend
	if len(tl.loader.StaticInputs) <= 0 && len(tl.loader.SourceFiles) <= 0 &&
		reflectx.StructOrStructPtrIsZero(tl.options) {
		return tl.loadFromBackend(ctx)
	}

	// Otherwise, attempt to load the static inputs from CLI and files
	inputs, err := targetloading.LoadStatic(tl.loader)

	// Handle the case where we couldn't
	if err != nil {
		return nil, err
	}

	// Build the list of targets that we should measure.
	var targets []model.ExperimentTarget
	for _, input := range inputs {
		targets = append(targets, &Target{
			Config: tl.options,
			URL:    input,
		})
	}
	return targets, nil
}

var defaultInput = []model.ExperimentTarget{
	//
	// https://dns.google/dns-query
	//
	// Measure HTTP/3 first and then HTTP/2 (see https://github.com/ooni/probe/issues/2675).
	//
	// Make sure we include the typical IP addresses for the domain.
	//
	&Target{
		URL: "https://dns.google/dns-query",
		Config: &Config{
			HTTP3Enabled: true,
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},
	&Target{
		URL: "https://dns.google/dns-query",
		Config: &Config{
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},

	// TODO(bassosimone,DecFox): before releasing, we need to either sync up
	// this list with ./internal/targetloader or implement a backend API.
}

func (tl *targetLoader) loadFromBackend(_ context.Context) ([]model.ExperimentTarget, error) {
	// TODO(https://github.com/ooni/probe/issues/1390): serve DNSCheck
	// inputs using richer input (aka check-in v2).
	return defaultInput, nil
}
