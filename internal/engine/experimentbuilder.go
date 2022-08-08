package engine

//
// ExperimentBuilder definition and implementation
//

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"

	"github.com/iancoleman/strcase"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// experimentBuilder implements ExperimentBuilder.
type experimentBuilder struct {
	// build is the constructor that build an experiment with the given config.
	build func(config interface{}) *experiment

	// callbacks contains callbacks for the new experiment.
	callbacks model.ExperimentCallbacks

	// config contains the experiment's config.
	config interface{}

	// inputPolicy contains the experiment's InputPolicy.
	inputPolicy model.InputPolicy

	// interruptible indicates whether the experiment is interruptible.
	interruptible bool
}

// Interruptible implements ExperimentBuilder.Interruptible.
func (b *experimentBuilder) Interruptible() bool {
	return b.interruptible
}

// InputPolicy implements ExperimentBuilder.InputPolicy.
func (b *experimentBuilder) InputPolicy() model.InputPolicy {
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

// Options implements ExperimentBuilder.Options.
func (b *experimentBuilder) Options() (map[string]model.ExperimentOptionInfo, error) {
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
func (b *experimentBuilder) setOptionBool(field reflect.Value, value any) error {
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
func (b *experimentBuilder) setOptionInt(field reflect.Value, value any) error {
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
func (b *experimentBuilder) setOptionString(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		field.SetString(v)
		return nil
	default:
		return fmt.Errorf("%w from a value of type %T", ErrCannotSetStringOption, value)
	}
}

// SetOptionAny implements ExperimentBuilder.SetOptionAny.
func (b *experimentBuilder) SetOptionAny(key string, value any) error {
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

// SetOptionsAny implements ExperimentBuilder.SetOptionsAny.
func (b *experimentBuilder) SetOptionsAny(options map[string]any) error {
	for key, value := range options {
		if err := b.SetOptionAny(key, value); err != nil {
			return err
		}
	}
	return nil
}

// SetCallbacks implements ExperimentBuilder.SetCallbacks.
func (b *experimentBuilder) SetCallbacks(callbacks model.ExperimentCallbacks) {
	b.callbacks = callbacks
}

// fieldbyname return v's field whose name is equal to the given key.
func (b *experimentBuilder) fieldbyname(v interface{}, key string) (reflect.Value, error) {
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

// NewExperiment creates the experiment
func (b *experimentBuilder) NewExperiment() model.Experiment {
	experiment := b.build(b.config)
	experiment.callbacks = b.callbacks
	return experiment
}

// canonicalizeExperimentName allows code to provide experiment names
// in a more flexible way, where we have aliases.
//
// Because we allow for uppercase experiment names for backwards
// compatibility with MK, we need to add some exceptions here when
// mapping (e.g., DNSCheck => dnscheck).
func canonicalizeExperimentName(name string) string {
	switch name = strcase.ToSnake(name); name {
	case "ndt_7":
		name = "ndt" // since 2020-03-18, we use ndt7 to implement ndt by default
	case "dns_check":
		name = "dnscheck"
	case "stun_reachability":
		name = "stunreachability"
	default:
	}
	return name
}

// newExperimentBuilder creates a new experimentBuilder instance.
func newExperimentBuilder(session *Session, name string) (*experimentBuilder, error) {
	factory := experimentsByName[canonicalizeExperimentName(name)]
	if factory == nil {
		return nil, fmt.Errorf("no such experiment: %s", name)
	}
	builder := factory(session)
	builder.callbacks = model.NewPrinterCallbacks(session.Logger())
	return builder, nil
}
