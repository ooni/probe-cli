package wireguard

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Target is a richer-input target that this experiment should measure.
type Target struct {
	// Options contains the configuration.
	Options *Config

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

// String implements [model.ExperimentTarget].
func (t *Target) String() string {
	return t.URL
}

// NewLoader constructs a new [model.ExperimentTargerLoader] instance.
//
// This function PANICS if options is not an instance of [*openvpn.Config].
func NewLoader(loader *targetloading.Loader, gopts any) model.ExperimentTargetLoader {
	// Panic if we cannot convert the options to the expected type.
	//
	// We do not expect a panic here because the type is managed by the registry package.
	options := gopts.(*Config)

	// Construct the proper loader instance.
	return &targetLoader{
		loader:  loader,
		options: options,
		session: loader.Session,
	}
}

// targetLoader loads targets for this experiment.
type targetLoader struct {
	loader  *targetloading.Loader
	options *Config
	session targetloading.Session
}

// Load implements model.ExperimentTargetLoader.
func (tl *targetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	// TODO(ainghazal): implement remote loading when backend is ready.

	// Attempt to load the static inputs from CLI and files
	inputs, err := targetloading.LoadStatic(tl.loader)

	// Handle the case where we couldn't load from CLI or files
	if err != nil {
		return nil, err
	}

	// Build the list of targets that we should measure.
	var targets []model.ExperimentTarget
	for _, input := range inputs {
		targets = append(targets, &Target{
			Options: tl.options,
			URL:     input,
		})
	}
	return targets, nil
}
