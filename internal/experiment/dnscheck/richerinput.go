package dnscheck

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Target is a richer-input target that this experiment should measure.
type Target struct {
	// input is the input.
	input string

	// options is the configuration.
	options *Config
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
	return t.input
}

// String implements [model.ExperimentTarget].
func (t *Target) String() string {
	return t.input
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
		loader:  loader,
		options: options,
	}
}

type targetLoader struct {
	loader  *targetloading.Loader
	options *Config
}

// Load implements model.ExperimentTargetLoader.
func (tl *targetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	// TODO(bassosimone): we need a way to know whether the options are empty!!!

	// If there's nothing to statically load fallback to the API
	if len(tl.loader.StaticInputs) <= 0 && len(tl.loader.SourceFiles) <= 0 {
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
			options: tl.options,
			input:   input,
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
		input: "https://dns.google/dns-query",
		options: &Config{
			HTTP3Enabled: true,
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},
	&Target{
		input: "https://dns.google/dns-query",
		options: &Config{
			DefaultAddrs: "8.8.8.8 8.8.4.4",
		},
	},

	// TODO(bassosimone): before merging, we need to reinstate the
	// whole list that we previously had in tree
}

func (tl *targetLoader) loadFromBackend(_ context.Context) ([]model.ExperimentTarget, error) {
	// TODO(https://github.com/ooni/probe/issues/1390): serve DNSCheck
	// inputs using richer input (aka check-in v2).
	return defaultInput, nil
}
