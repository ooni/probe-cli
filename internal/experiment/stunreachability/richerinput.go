package stunreachability

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/experimentconfig"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/reflectx"
	"github.com/ooni/probe-cli/v3/internal/stuninput"
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

// NewLoader constructs a new [model.ExperimentTargetLoader] instance.
//
// This function PANICS if options is not an instance of [*stunreachability.Config].
func NewLoader(loader *targetloading.Loader, gopts any) model.ExperimentTargetLoader {
	// Panic if we cannot convert the options to the expected type.
	//
	// We do not expect a panic here because the type is managed by the registry package.
	options := gopts.(*Config)

	return &targetLoader{
		defaultInput: defaultInputTargets,
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
	// If inputs and files are all empty and there are no options, use the default input.
	if len(tl.loader.StaticInputs) <= 0 && len(tl.loader.SourceFiles) <= 0 &&
		reflectx.StructOrStructPtrIsZero(tl.options) {
		return tl.loadFromBackend(ctx)
	}

	// Otherwise, attempt to load the static inputs from CLI and files.
	inputs, err := targetloading.LoadStatic(tl.loader)
	if err != nil {
		return nil, err
	}

	// Build the list of targets. Each target starts from the experiment-wide
	// config with any per-input inputs_extra overlaid on top.
	var targets []model.ExperimentTarget
	for i, input := range inputs {
		targets = append(targets, &Target{
			Config: targetloading.PerInputConfig(tl.loader, tl.options, i),
			URL:    input,
		})
	}
	return targets, nil
}

// defaultInputTargets is the built-in list of STUN endpoints to measure when no
// input is provided. Each entry carries a default per-input config.
var defaultInputTargets = stunReachabilityDefaultTargets()

// stunReachabilityDefaultTargets builds the default target list from the
// built-in list of STUN endpoints.
func stunReachabilityDefaultTargets() []model.ExperimentTarget {
	var targets []model.ExperimentTarget
	for _, url := range stuninput.AsnStunReachabilityInput() {
		targets = append(targets, &Target{
			Config: &Config{},
			URL:    url,
		})
	}
	return targets
}

func (tl *targetLoader) loadFromBackend(_ context.Context) ([]model.ExperimentTarget, error) {
	// TODO(https://github.com/ooni/probe/issues/2557): serve STUNReachability
	// inputs using richer input (aka check-in v2).
	return tl.defaultInput, nil
}
