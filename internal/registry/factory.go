package registry

//
// Factory for constructing experiments.
//

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strconv"

	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/experimentname"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Factory allows to construct an experiment measurer.
type Factory struct {
	// build is the constructor that build an experiment with the given config.
	build func(config interface{}) model.ExperimentMeasurer

	// canonicalName is the canonical name of the experiment.
	canonicalName string

	// config contains the experiment's config.
	config any

	// enabledByDefault indicates whether this experiment is enabled by default.
	enabledByDefault bool

	// inputPolicy contains the experiment's InputPolicy.
	inputPolicy model.InputPolicy

	// interruptible indicates whether the experiment is interruptible.
	interruptible bool

	// newLoader is the OPTIONAL function to create a new loader.
	newLoader func(config *targetloading.Loader, options any) model.ExperimentTargetLoader
}

// Session is the session definition according to this package.
type Session = model.ExperimentTargetLoaderSession

// NewTargetLoader creates a new [model.ExperimentTargetLoader] instance.
func (b *Factory) NewTargetLoader(config *model.ExperimentTargetLoaderConfig) model.ExperimentTargetLoader {
	// Construct the default loader used in the non-richer input case.
	loader := &targetloading.Loader{
		CheckInConfig:  config.CheckInConfig, // OPTIONAL
		ExperimentName: b.canonicalName,
		InputPolicy:    b.inputPolicy,
		Logger:         config.Session.Logger(),
		Session:        config.Session,
		StaticInputs:   config.StaticInputs,
		SourceFiles:    config.SourceFiles,
	}

	// If an experiment implements richer input, it will use its custom loader
	// that will use experiment specific policy for loading targets.
	if b.newLoader != nil {
		return b.newLoader(loader, b.config)
	}

	// Otherwise just return the default loader.
	return loader
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
	// create the result value
	result := make(map[string]model.ExperimentOptionInfo)

	// make sure we're dealing with a pointer
	ptrinfo := reflect.ValueOf(b.config)
	if ptrinfo.Kind() != reflect.Ptr {
		return nil, ErrConfigIsNotAStructPointer
	}

	// obtain information about the value and its type
	valueinfo := ptrinfo.Elem()
	typeinfo := valueinfo.Type()

	// make sure we're dealing with a struct
	if typeinfo.Kind() != reflect.Struct {
		return nil, ErrConfigIsNotAStructPointer
	}

	// cycle through the fields
	for i := 0; i < typeinfo.NumField(); i++ {
		fieldType, fieldValue := typeinfo.Field(i), valueinfo.Field(i)

		// do not include private fields into our list of fields
		if !fieldType.IsExported() {
			continue
		}

		// skip fields that are missing an `ooni` tag
		docs := fieldType.Tag.Get("ooni")
		if docs == "" {
			continue
		}

		// create a description of this field
		result[fieldType.Name] = model.ExperimentOptionInfo{
			Doc:   docs,
			Type:  fieldType.Type.String(),
			Value: fieldValue.Interface(),
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

// With JSON we're limited by the 52 bits in the mantissa
const (
	jsonMaxInteger = 1<<53 - 1
	jsonMinInteger = -1<<53 + 1
)

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
	case float64:
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return fmt.Errorf("%w from: %v", ErrCannotSetIntegerOption, value)
		}
		if math.Trunc(v) != v {
			return fmt.Errorf("%w from: %v", ErrCannotSetIntegerOption, value)
		}
		if v > jsonMaxInteger || v < jsonMinInteger {
			return fmt.Errorf("%w from: %v", ErrCannotSetIntegerOption, value)
		}
		field.SetInt(int64(v))
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

// SetOptionsJSON unmarshals the given [json.RawMessage] inside
// the experiment specific configuration.
func (b *Factory) SetOptionsJSON(value json.RawMessage) error {
	// handle the case where the options are empty
	if len(value) <= 0 {
		return nil
	}

	// otherwise unmarshal into the configuration, which we assume
	// to be a pointer to a structure.
	return json.Unmarshal(value, b.config)
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

// NewExperimentMeasurer creates a new [model.ExperimentMeasurer] instance.
func (b *Factory) NewExperimentMeasurer() model.ExperimentMeasurer {
	return b.build(b.config)
}

// ErrNoSuchExperiment indicates a given experiment does not exist.
var ErrNoSuchExperiment = errors.New("no such experiment")

// ErrRequiresForceEnable is returned for experiments that are not enabled by default and are also
// not enabled by the most recent check-in API call.
var ErrRequiresForceEnable = errors.New("experiment not enabled by check-in API")

const experimentDisabledByCheckInWarning = `We disabled the '%s' nettest. This usually happens in these cases:

1. we just added the nettest to ooniprobe and we have not enabled it yet;

2. the nettest is flaky and we are working on a fix;

3. you ran Web Connectivity more than 24h ago, hence your check-in cache is stale.

The last case is a known limitation in ooniprobe 3.19 that we will fix in a subsequent
release of ooniprobe by changing the nettests startup logic.

If you really want to run this nettest, there is a way forward. You need to set the
OONI_FORCE_ENABLE_EXPERIMENT=1 environment variable. On a Unix like system, use:

    export OONI_FORCE_ENABLE_EXPERIMENT=1

on Windows use:

    set OONI_FORCE_ENABLE_EXPERIMENT=1

Re-running ooniprobe once you have set the environment variable would cause the
disabled nettest to run. Please, note that we usually have good reasons for disabling
nettests, including the following reasons:

* making sure that we gradually introduce new nettests to all users by first introducing
them to a few users and monitoring whether they're working as intended;

* avoid polluting our measurements database with measurements produced by experiments
that currently produce false positives or other data quality issues.
`

// OONI_FORCE_ENABLE_EXPERIMENT is the name of the environment variable you should set to "1"
// to bypass the algorithm preventing disabled by default experiments to be instantiated.
const OONI_FORCE_ENABLE_EXPERIMENT = "OONI_FORCE_ENABLE_EXPERIMENT"

// NewFactory creates a new Factory instance.
func NewFactory(name string, kvStore model.KeyValueStore, logger model.Logger) (*Factory, error) {
	// Make sure we are deadling with the canonical experiment name. Historically MK used
	// names such as WebConnectivity and we want to continue supporting this use case.
	name = experimentname.Canonicalize(name)

	// Handle A/B testing where we dynamically choose LTE for some users. The current policy
	// only relates to a few users to collect data.
	//
	// TODO(https://github.com/ooni/probe/issues/2555): perform the actual comparison
	// and improve the LTE implementation so that we can always use it. See the actual
	// issue test for additional details on this planned A/B test.
	switch {
	case name == "web_connectivity" && checkincache.GetFeatureFlag(kvStore, "webconnectivity_0.5", false):
		// use LTE rather than the normal webconnectivity when the
		// feature flag has been set through the check-in API
		logger.Infof("using webconnectivity LTE")
		name = "web_connectivity@v0.5"

	default:
		// nothing
	}

	// Obtain the factory for the canonical name.
	ff := AllExperiments[name]
	if ff == nil {
		return nil, fmt.Errorf("%w: %s", ErrNoSuchExperiment, name)
	}
	factory := ff()

	// Some experiments are not enabled by default. To enable them we use
	// the cached check-in response or an environment variable.
	//
	// Note: check-in flags expire after 24h.
	//
	//
	//
	if factory.enabledByDefault {
		if !checkincache.ExperimentEnabled(kvStore, name, true) {
			return nil, fmt.Errorf("%s: %w", name, ErrRequiresForceEnable)
		}
		return factory, nil
	}
	if os.Getenv(OONI_FORCE_ENABLE_EXPERIMENT) == "1" {
		return factory, nil // enabled by environment variable
	}
	if checkincache.ExperimentEnabled(kvStore, name, false) {
		return factory, nil // enabled by check-in
	}

	logger.Warnf(experimentDisabledByCheckInWarning, name)
	return nil, fmt.Errorf("%s: %w", name, ErrRequiresForceEnable)
}
