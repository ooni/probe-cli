package registryx

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/database"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/spf13/cobra"
)

// Factory is a forwarder for the respective experiment's main
type Factory struct {
	// Main calls the experiment.Main functions
	Main func(ctx context.Context, sess *engine.Session, db *database.DatabaseProps) error

	//
	Oonirun func(ctx context.Context, sess *engine.Session, inputs []string,
		args map[string]any, extraOptions map[string]any, db *database.DatabaseProps) error

	// BuildFlags initializes the experiment specific flags
	BuildFlags func(experimentName string, rootCmd *cobra.Command)
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

// options returns the options exposed by this experiment.
func options(config any) (map[string]model.ExperimentOptionInfo, error) {
	result := make(map[string]model.ExperimentOptionInfo)
	ptrinfo := reflect.ValueOf(config)
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
func setOptionBool(field reflect.Value, value any) error {
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
func setOptionInt(field reflect.Value, value any) error {
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
func setOptionString(field reflect.Value, value any) error {
	switch v := value.(type) {
	case string:
		field.SetString(v)
		return nil
	default:
		return fmt.Errorf("%w from a value of type %T", ErrCannotSetStringOption, value)
	}
}

// setOptionAny sets an option given any value.
func setOptionAny(config any, key string, value any) error {
	field, err := fieldbyname(config, key)
	if err != nil {
		return err
	}
	switch field.Kind() {
	case reflect.Int64:
		return setOptionInt(field, value)
	case reflect.Bool:
		return setOptionBool(field, value)
	case reflect.String:
		return setOptionString(field, value)
	default:
		return fmt.Errorf("%w: %T", ErrUnsupportedOptionType, value)
	}
}

// SetOptionsAny calls SetOptionAny for each entry inside [options].
func setOptionsAny(config any, options map[string]any) error {
	for key, value := range options {
		if err := setOptionAny(config, key, value); err != nil {
			return err
		}
	}
	return nil
}

// fieldbyname return v's field whose name is equal to the given key.
func fieldbyname(v interface{}, key string) (reflect.Value, error) {
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

func documentationForOptions(name string, config any) string {
	var sb strings.Builder
	options, err := options(config)
	if err != nil || len(options) < 1 {
		return ""
	}
	fmt.Fprint(&sb, "Pass KEY=VALUE options to the experiment. Available options:\n")
	for name, info := range options {
		if info.Doc == "" {
			continue
		}
		fmt.Fprintf(&sb, "\n")
		fmt.Fprintf(&sb, "  -O, --option %s=<%s>\n", name, info.Type)
		fmt.Fprintf(&sb, "      %s\n", info.Doc)
	}
	return sb.String()
}
