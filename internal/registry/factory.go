package registry

//
// Factory for constructing experiments.
//

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/strcasex"
)

// Factory allows to construct an experiment measurer.
type Factory struct {
	// build is the constructor that build an experiment with the given config.
	build func(config interface{}) model.ExperimentMeasurer

	// config contains the experiment's config.
	config any

	// enabledByDefault indicates whether this experiment is enabled by default.
	enabledByDefault bool

	// inputPolicy contains the experiment's InputPolicy.
	inputPolicy model.InputPolicy

	// interruptible indicates whether the experiment is interruptible.
	interruptible bool
}

// Interruptible returns whether the experiment is interruptible.
func (b *Factory) Interruptible() bool {
	return b.interruptible
}

// InputPolicy returns the experiment's InputPolicy.
func (b *Factory) InputPolicy() model.InputPolicy {
	return b.inputPolicy
}

var (
	// ErrConfigIsNotAStructPointer indicates we expected a pointer to struct.
	ErrConfigIsNotAStructPointer = errors.New("config is not a struct pointer")

	// ErrNoSuchField indicates there's no field with the given name.
	ErrNoSuchField = errors.New("no such field")

	// ErrCannotSetIntegerOption means SetOptionAny couldn't set an integer option.
	ErrCannotSetIntegerOption = errors.New("cannot set integer option")

	// ErrInvalidStringRepresentationOfBool indicates the string you passed
	// to SetOptionaAny is not a valid string representation of a bool.
	ErrInvalidStringRepresentationOfBool = errors.New("invalid string representation of bool")

	// ErrCannotSetBoolOption means SetOptionAny couldn't set a bool option.
	ErrCannotSetBoolOption = errors.New("cannot set bool option")

	// ErrCannotSetStringOption means SetOptionAny couldn't set a string option.
	ErrCannotSetStringOption = errors.New("cannot set string option")

	// ErrUnsupportedOptionType means we don't support the type passed to
	// the SetOptionAny method as an opaque any type.
	ErrUnsupportedOptionType = errors.New("unsupported option type")
)

// Options returns the options exposed by this experiment.
func (b *Factory) Options() (map[string]model.ExperimentOptionInfo, error) {
	result := make(map[string]model.ExperimentOptionInfo)
	ptrinfo := reflect.ValueOf(b.config)
	if ptrinfo.Kind() != reflect.Ptr {
		return nil, ErrConfigIsNotAStructPointer
	}
	structinfo := ptrinfo.Elem().Type()
	if structinfo.Kind() != reflect.Struct {
		return nil, ErrConfigIsNotAStructPointer
	}
	for i := 0; i < structinfo.NumField(); i++ {
		field := structinfo.Field(i)
		result[field.Name] = model.ExperimentOptionInfo{
			Doc:  field.Tag.Get("ooni"),
			Type: field.Type.String(),
		}
	}
	return result, nil
}

// setOptionBool sets a bool option.
func (b *Factory) setOptionBool(field reflect.Value, value any) error {
	switch v := value.(type) {
	case bool:
		field.SetBool(v)
		return nil
	case string:
		if v != "true" && v != "false" {
			return fmt.Errorf("%w: %s", ErrInvalidStringRepresentationOfBool, v)
		}
		field.SetBool(v == "true")
		return nil
	default:
		return fmt.Errorf("%w from a value of type %T", ErrCannotSetBoolOption, value)
	}
}

// setOptionInt sets an int option
func (b *Factory) setOptionInt(field reflect.Value, value any) error {
	switch v := value.(type) {
	case int64:
		field.SetInt(v)
		return nil
	case int32:
		field.SetInt(int64(v))
		return nil
	case int16:
		field.SetInt(int64(v))
		return nil
	case int8:
		field.SetInt(int64(v))
		return nil
	case int:
		field.SetInt(int64(v))
		return nil
	case string:
		number, err := strconv.ParseInt(v, 10, 64)
		if err != nil {
			return fmt.Errorf("%w: %s", ErrCannotSetIntegerOption, err.Error())
		}
		field.SetInt(number)
		return nil
	default:
		return fmt.Errorf("%w from a value of type %T", ErrCannotSetIntegerOption, value)
	}
}

// setOptionString sets a string option
func (b *Factory) setOptionString(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		field.SetString(v)
		return nil
	default:
		return fmt.Errorf("%w from a value of type %T", ErrCannotSetStringOption, value)
	}
}

// SetOptionAny sets an option given any value.
func (b *Factory) SetOptionAny(key string, value any) error {
	field, err := b.fieldbyname(b.config, key)
	if err != nil {
		return err
	}
	switch field.Kind() {
	case reflect.Int64:
		return b.setOptionInt(field, value)
	case reflect.Bool:
		return b.setOptionBool(field, value)
	case reflect.String:
		return b.setOptionString(field, value)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedOptionType, value)
	}
}

// SetOptionsAny calls SetOptionAny for each entry inside [options].
func (b *Factory) SetOptionsAny(options map[string]any) error {
	for key, value := range options {
		if err := b.SetOptionAny(key, value); err != nil {
			return err
		}
	}
	return nil
}

// fieldbyname return v's field whose name is equal to the given key.
func (b *Factory) fieldbyname(v interface{}, key string) (reflect.Value, error) {
	// See https://stackoverflow.com/a/6396678/4354461
	ptrinfo := reflect.ValueOf(v)
	if ptrinfo.Kind() != reflect.Ptr {
		return reflect.Value{}, fmt.Errorf("%w but a %T", ErrConfigIsNotAStructPointer, v)
	}
	structinfo := ptrinfo.Elem()
	if structinfo.Kind() != reflect.Struct {
		return reflect.Value{}, fmt.Errorf("%w but a %T", ErrConfigIsNotAStructPointer, v)
	}
	field := structinfo.FieldByName(key)
	if !field.IsValid() || !field.CanSet() {
		return reflect.Value{}, fmt.Errorf("%w: %s", ErrNoSuchField, key)
	}
	return field, nil
}

// NewExperimentMeasurer creates the experiment
func (b *Factory) NewExperimentMeasurer() model.ExperimentMeasurer {
	return b.build(b.config)
}

// CanonicalizeExperimentName allows code to provide experiment names
// in a more flexible way, where we have aliases.
//
// Because we allow for uppercase experiment names for backwards
// compatibility with MK, we need to add some exceptions here when
// mapping (e.g., DNSCheck => dnscheck).
func CanonicalizeExperimentName(name string) string {
	switch name = strcasex.ToSnake(name); name {
	case "ndt_7":
		name = "ndt" // since 2020-03-18, we use ndt7 to implement ndt by default
	case "dns_check":
		name = "dnscheck"
	case "stun_reachability":
		name = "stunreachability"
	case "web_connectivity@v_0_5":
		name = "web_connectivity@v0.5"
	default:
	}
	return name
}

// ErrNoSuchExperiment indicates a given experiment does not exist.
var ErrNoSuchExperiment = errors.New("no such experiment")

// ErrRequiresForceEnable is returned for experiments that are not enabled by default and are also
// not enabled by the most recent check-in API call.
var ErrRequiresForceEnable = errors.New("experiment not enabled by check-in API")

const experimentDisabledByCheckInWarning = `experiment '%s' is not enabled by default and the
most recent check-in API call did not enable this experiment as well. You can bypass this restriction
by setting the OONI_FORCE_ENABLE_EXPERIMENT environment variable to the string "1". On Unix like
systems, you can use 'export OONI_FORCE_ENABLE_EXPERIMENT=1' to set this environment variable.`

const experimentForceEnableEnvVar = "OONI_FORCE_ENABLE_EXPERIMENT"

// NewFactory creates a new Factory instance.
func NewFactory(name string, kvStore model.KeyValueStore, logger model.Logger) (*Factory, error) {
	// Make sure we are deadling with the canonical experiment name. Historically MK used
	// names such as WebConnectivity and we want to continue supporting this use case.
	name = CanonicalizeExperimentName(name)

	// Handle A/B testing where we dynamically choose LTE for some users. The current policy
	// only relates to a few users to collect data.
	//
	// TODO(https://github.com/ooni/probe/issues/2555): perform the actual comparison
	// and improve the LTE implementation so that we can always use it. See the actual
	// issue test for additional details on this planned A/B test.
	name = CanonicalizeExperimentName(name)
	switch {
	case name == "web_connectivity" && checkincache.GetFeatureFlag(kvStore, "webconnectivity_0.5"):
		// use LTE rather than the normal webconnectivity when the
		// feature flag has been set through the check-in API
		logger.Infof("using webconnectivity LTE")
		name = "web_connectivity@v0.5"

	default:
		// nothing
	}

	// Obtain the factory for the canonical name.
	factory := AllExperiments[name]
	if factory == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoSuchExperiment, name)
	}

	// Some experiments are not enabled by default. To enable them we use
	// the cached check-in response or an environment variable.
	//
	// Note: check-in flags expire after 24h.
	//
	// TODO(https://github.com/ooni/probe/issues/2554): we need to restructure
	// of we run experiments to make sure check-in flags are always fresh.
	if factory.enabledByDefault {
		return factory, nil // enabled by default
	}
	if os.Getenv(experimentForceEnableEnvVar) == "1" {
		return factory, nil // enabled by environment variable
	}
	if checkincache.ExperimentEnabled(kvStore, name) {
		return factory, nil // enabled by check-in
	}

	logger.Warnf(experimentDisabledByCheckInWarning, name)
	return nil, fmt.Errorf("%s: %w", name, ErrRequiresForceEnable)
}
