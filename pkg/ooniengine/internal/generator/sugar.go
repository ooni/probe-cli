package main

import "fmt"

//
// Syntactic sugar to avoid repeating ourselves
// when creating specific events and tasks.
//

// Suffix added to all the constants containing the name of events
const EventNameSuffix = "EventName"

// Suffix added to all the structs containing the value of events
const EventValueSuffix = "EventValue"

// registerNewEvent registers a new event with the given name.
//
// Arguments:
//
// - docs contains documentation for the struct;
//
// - name is the struct name WITHOUT the "Event" suffix;
//
// - fields contains the struct fields.
//
// This function adds the struct to the global ABI singleton.
//
// Returns the name of the newly added structure.
func registerNewEvent(docs, name string, fields ...StructField) string {
	OONIEngine.Constants = append(OONIEngine.Constants, Constant{
		Docs: []string{
			fmt.Sprintf("Name of the %s event", name),
		},
		Name:  name + EventNameSuffix,
		Value: name,
	})
	return registerNewStructEx(BaseEvent, docs, name+EventValueSuffix, fields...)
}

// Suffix added to all structs that are configs
const ConfigSuffix = "Config"

// registerNewConfig registers a new config with the given name.
//
// Arguments:
//
// - docs contains documentation for the struct;
//
// - name is the struct name WITHOUT the "Config" suffix;
//
// - fields contains the struct fields.
//
// This function adds the struct to the global ABI singleton.
//
// Returns the name of the newly added structure.
func registerNewConfig(docs, name string, fields ...StructField) string {
	return registerNewStructEx(BaseConfig, docs, name+ConfigSuffix, fields...)
}

// registerNewStruct registers a new struct with the given name.
//
// Arguments:
//
// - docs contains documentation for the struct;
//
// - name is the struct name;
//
// - fields contains the struct fields.
//
// This function adds the struct to the global ABI singleton.
//
// Returns the name of the newly added structure.
func registerNewStruct(docs, name string, fields ...StructField) string {
	return registerNewStructEx("", docs, name, fields...)
}

// registerNewStructEx is like registerNewStruct except that
// it also allows you to specify the superclass.
func registerNewStructEx(baseClass, docs, name string, fields ...StructField) string {
	OONIEngine.Structs = append(OONIEngine.Structs, Struct{
		Docs:       []string{docs},
		Name:       name,
		Fields:     fields,
		Superclass: baseClass,
	})
	return name
}
