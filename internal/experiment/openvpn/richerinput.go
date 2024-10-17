package openvpn

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/experimentconfig"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// providerAuthentication is a map so that we know which kind of credentials we
// need to fill in the openvpn options for each known provider.
var providerAuthentication = map[string]AuthMethod{
	"riseupvpn":     AuthCertificate,
	"tunnelbearvpn": AuthUserPass,
	"surfsharkvpn":  AuthUserPass,
}

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
func (t *Target) Options() (options []string) {
	return experimentconfig.DefaultOptionsSerializer(t.Config)
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
	// First, attempt to load the static inputs from CLI and files
	inputs, err := targetloading.LoadStatic(tl.loader)
	// Handle the case where we couldn't load from CLI or files (fallthru)
	if err != nil {
		tl.loader.Logger.Warnf("Error loading OpenVPN targets from cli")
	}

	// Build the list of targets that we should measure.
	var targets []model.ExperimentTarget
	for _, input := range inputs {
		targets = append(targets, &Target{
			Config: tl.options,
			URL:    input,
		})
	}
	if len(targets) > 0 {
		return targets, nil
	}

	// Return the hardcoded endpoints.
	return tl.loadFromDefaultEndpoints()
}

func (tl *targetLoader) loadFromDefaultEndpoints() ([]model.ExperimentTarget, error) {
	targets := []model.ExperimentTarget{}

	addrs, err := resolveOONIAddresses(tl.session.Logger())
	if err != nil {
		return targets, err
	}

	tl.loader.Logger.Warnf("Picking from default OpenVPN endpoints")
	if inputs, err := pickOONIOpenVPNTargets(addrs); err == nil {
		for _, url := range inputs {
			targets = append(targets,
				&Target{
					Config: pickFromDefaultOONIOpenVPNConfig(),
					URL:    url,
				})
		}
	}
	return targets, nil
}
