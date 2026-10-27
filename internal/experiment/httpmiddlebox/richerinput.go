package httpmiddlebox

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
// This function PANICS if options is not an instance of [*httpmiddlebox.Config].
func NewLoader(loader *targetloading.Loader, gopts any) model.ExperimentTargetLoader {
	// Panic if we cannot convert the options to the expected type.
	//
	// We do not expect a panic here because the type is managed by the registry package.
	options := gopts.(*Config)

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
	if len(tl.loader.StaticInputs) == 0 && len(tl.loader.SourceFiles) == 0 {
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
	for i, input := range inputs {
		targets = append(targets, &Target{
			Config:   targetloading.PerInputConfig(tl.loader, tl.options, i),
			URL:      input,
			category: model.DefaultCategoryCode,
			country:  model.DefaultCountryCode,
		})
	}
	return targets, nil
}

var defaultInput = []model.ExperimentTarget{
	&Target{
		URL:      "http://facebook.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://instagram.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://twitter.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://youtube.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://xvideos.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://fbcdn.net",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://rfi.fr",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://grindr.com",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://torproject.org",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
	&Target{
		URL:      "http://anonymouse.org",
		Config:   &Config{},
		category: model.DefaultCategoryCode,
		country:  model.DefaultCountryCode,
	},
}

func (tl *targetLoader) loadFromBackend(_ context.Context) ([]model.ExperimentTarget, error) {
	// TODO(https://github.com/ooni/probe/issues/1390): serve DNSCheck
	// inputs using richer input (aka check-in v2).
	return tl.defaultInput, nil
}
