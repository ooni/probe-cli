package main

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

//
// Data model
//

// Constant is a constant.
type Constant struct {
	// Docs contains the constant's docs.
	Docs []string

	// Name is the constant's name.
	Name string

	// Value is the constant's value.
	Value string
}

// Struct is a struct.
type Struct struct {
	// Docs contains the struct's docs.
	Docs []string

	// Name is the struct's name.
	Name string

	// Fields contains the struct's fields.
	Fields []StructField

	// Superclass is the superclass.
	Superclass string
}

// Type is a type of something.
type Type string

const (
	// TypeString is the string type.
	TypeString = Type("::string")

	// TypeInt64 is the int64 type.
	TypeInt64 = Type("::int64")

	// TypeListString is a []string.
	TypeListString = Type("::list<string>")

	// TypeMapStringString is a map[string]string.
	TypeMapStringString = Type("::map<string, string>")

	// TypeMapStringAny is a map[string]any.
	TypeMapStringAny = Type("::map<string, any>")

	// TypeBool is the type of a bool.
	TypeBool = Type("::bool")

	// TypeFloat64 is a float64 number.
	TypeFloat64 = Type("::float64")
)

// StructField is a struct field.
type StructField struct {
	// Docs contains the field's docs.
	Docs []string

	// Name is the field's name.
	Name string

	// Type is the field's type.
	Type Type
}

// Task is a task you can run from dart.
type Task struct {
	// Name is the task's name.
	Name string

	// Config contains the name of the config struct.
	Config string
}

// ABI describes the ABI.
type ABI struct {
	// Constants contains the ABI constants.
	Constants []Constant

	// Structs contains the ABI structs.
	Structs []Struct

	// Tasks contains the ABI tasks.
	Tasks []Task
}

// OONIEngine is the OONIEngine ABI.
var OONIEngine = &ABI{}

// BaseEvent is the base class for all events
const BaseEvent = "BaseEvent"

// BaseConfig is the base class for all configs
const BaseConfig = "BaseConfig"

func init() {
	OONIEngine.Structs = append(OONIEngine.Structs, Struct{
		Docs: []string{
			"Base class for all events.",
		},
		Name:       BaseEvent,
		Fields:     []StructField{},
		Superclass: "",
	})

	OONIEngine.Structs = append(OONIEngine.Structs, Struct{
		Docs: []string{
			"Base class for all configs.",
		},
		Name:       BaseConfig,
		Fields:     []StructField{},
		Superclass: "",
	})
}

// Returns whether this struct is an event
func (s *Struct) isEvent() bool {
	return s.Superclass == BaseEvent
}

// Converts the struct name of an event to the event name.
func (s *Struct) toEventName() string {
	runtimex.PanicIfFalse(s.isEvent(), "not an event")
	return strings.ReplaceAll(s.Name, EventValueSuffix, "")
}
