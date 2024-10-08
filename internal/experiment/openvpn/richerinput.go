package openvpn

import (
	"context"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/experimentconfig"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/reflectx"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// defaultProvider is the provider we will request from API in case we got no provider set
// in the CLI options.
var defaultProvider = "riseupvpn"

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

	// If inputs and files are all empty and there are no options, let's use the backend
	if len(tl.loader.StaticInputs) <= 0 && len(tl.loader.SourceFiles) <= 0 &&
		reflectx.StructOrStructPtrIsZero(tl.options) {
		targets, err := tl.loadFromBackend(ctx)
		if err == nil {
			return targets, nil
		}
		tl.loader.Logger.Warnf("Error fetching OpenVPN targets from backend")
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

func makeTargetListPerProtocol(cc string, num int) []model.ExperimentTarget {
	targets := []model.ExperimentTarget{}
	var reverse bool
	switch num {
	case 1, 2:
		// for single or few picks, we start the list in the natural order
		reverse = false
	default:
		// for multiple picks, we start the list from the bottom, so that we can lookup
		// custom country campaigns first.
		reverse = true
	}
	if inputsUDP, err := pickOONIOpenVPNTargets("udp", cc, num, reverse); err == nil {
		for _, t := range inputsUDP {
			targets = append(targets,
				&Target{
					Config: pickFromDefaultOONIOpenVPNConfig(),
					URL:    t,
				})
		}
	}
	if inputsTCP, err := pickOONIOpenVPNTargets("tcp", cc, num, reverse); err == nil {
		for _, t := range inputsTCP {
			targets = append(targets,
				&Target{
					Config: pickFromDefaultOONIOpenVPNConfig(),
					URL:    t,
				})
		}
	}
	return targets
}

func (tl *targetLoader) loadFromDefaultEndpoints() ([]model.ExperimentTarget, error) {
	cc := tl.session.ProbeCC()

	tl.loader.Logger.Warnf("Using default OpenVPN endpoints")
	tl.loader.Logger.Warnf("Picking endpoints for %s", cc)

	var targets []model.ExperimentTarget
	switch cc {
	case "RU", "CN", "IR", "EG", "NL":
		// we want to cover all of our bases for a few interest countries
		targets = makeTargetListPerProtocol(cc, 20)
	default:
		targets = makeTargetListPerProtocol(cc, 1)
	}
	return targets, nil
}

func (tl *targetLoader) loadFromBackend(ctx context.Context) ([]model.ExperimentTarget, error) {
	if tl.options.Provider == "" {
		tl.options.Provider = defaultProvider
	}

	targets := make([]model.ExperimentTarget, 0)
	provider := tl.options.Provider

	apiConfig, err := tl.session.FetchOpenVPNConfig(ctx, provider, tl.session.ProbeCC())
	if err != nil {
		tl.session.Logger().Warnf("Cannot fetch openvpn config: %v", err)
		return nil, err
	}

	auth, ok := providerAuthentication[provider]
	if !ok {
		return nil, fmt.Errorf("%w: unknown authentication for provider %s", targetloading.ErrInvalidInput, provider)
	}

	for _, input := range apiConfig.Inputs {
		config := &Config{
			Auth:   "SHA512",
			Cipher: "AES-256-GCM",
		}
		switch auth {
		case AuthCertificate:
			config.SafeCA = apiConfig.Config.CA
			config.SafeCert = apiConfig.Config.Cert
			config.SafeKey = apiConfig.Config.Key
		case AuthUserPass:
			// TODO(ainghazal): implement (surfshark, etc)
		}
		targets = append(targets, &Target{
			URL:    input,
			Config: config,
		})
	}

	return targets, nil
}
