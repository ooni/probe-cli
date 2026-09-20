package webconnectivitylte

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/experimentconfig"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Target is a richer-input target that this experiment should measure.
type Target struct {
	// Config contains the configuration.
	Config *Config

	// URL is the input URL.
	URL string

	// category is the citizenlab category code for this URL (possibly empty).
	category string

	// country is the country code for this URL (possibly empty).
	country string
}

var _ model.ExperimentTarget = &Target{}

// Category implements [model.ExperimentTarget].
func (t *Target) Category() string {
	if t.category != "" {
		return t.category
	}
	return model.DefaultCategoryCode
}

// Country implements [model.ExperimentTarget].
func (t *Target) Country() string {
	if t.country != "" {
		return t.country
	}
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

// NewLoader constructs a new [model.ExperimentTargetLoader] instance.
//
// This function PANICS if options is not an instance of [*webconnectivitylte.Config].
func NewLoader(loader *targetloading.Loader, gopts any) model.ExperimentTargetLoader {
	// Panic if we cannot convert the options to the expected type.
	//
	// We do not expect a panic here because the type is managed by the registry package.
	options := gopts.(*Config)

	return &targetLoader{
		loader:  loader,
		options: options,
	}
}

// targetLoader loads targets for this experiment.
type targetLoader struct {
	loader  *targetloading.Loader
	options *Config
}

// Load implements model.ExperimentTargetLoader.
func (tl *targetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	// Delegate to the underlying loader so we honour the InputOrQueryBackend policy.
	inputs, err := tl.loader.Load(ctx)
	if err != nil {
		return nil, err
	}

	// Wrap each input into a richer-input target, preserving the category and
	// country codes and overlaying any per-input inputs_extra onto the config.
	var targets []model.ExperimentTarget
	for i, input := range inputs {
		targets = append(targets, &Target{
			Config:   targetloading.PerInputConfig(tl.loader, tl.options, i),
			URL:      input.Input(),
			category: input.Category(),
			country:  input.Country(),
		})
	}
	return targets, nil
}
