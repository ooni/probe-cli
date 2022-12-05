// Package openapi converts httpapi Specs to .
package openapi

import "github.com/ooni/probe-cli/v3/internal/httpapi"

// Schema is the schema of a specific parameter or
// or the schema used by the response body
type Schema struct {
	Properties map[string]*Schema `json:"properties,omitempty"`
	Items      *Schema            `json:"items,omitempty"`
	Type       string             `json:"type"`
}

// Parameter describes an input parameter, which could be in the
// URL path, in the query string, or in the request body
type Parameter struct {
	In       string  `json:"in"`
	Name     string  `json:"name"`
	Required bool    `json:"required,omitempty"`
	Schema   *Schema `json:"schema,omitempty"`
	Type     string  `json:"type,omitempty"`
}

// Body describes a response body
type Body struct {
	Description interface{} `json:"description,omitempty"`
	Schema      *Schema     `json:"schema"`
}

// Responses describes the possible responses
type Responses struct {
	Successful Body `json:"200"`
}

// RoundTrip describes an HTTP round trip with a given method and path
type RoundTrip struct {
	Consumes   []string     `json:"consumes,omitempty"`
	Produces   []string     `json:"produces,omitempty"`
	Parameters []*Parameter `json:"parameters,omitempty"`
	Responses  *Responses   `json:"responses,omitempty"`
}

// Path describes a path served by the API
type Path struct {
	Get  *RoundTrip `json:"get,omitempty"`
	Post *RoundTrip `json:"post,omitempty"`
}

// Info contains info about the Info
type Info struct {
	Title   string `json:"title"`
	Version string `json:"version"`
}

// Swagger is the toplevel structure
type Swagger struct {
	Swagger  string           `json:"swagger"`
	Info     Info             `json:"info"`
	Host     string           `json:"host"`
	BasePath string           `json:"basePath"`
	Schemes  []string         `json:"schemes"`
	Paths    map[string]*Path `json:"paths"`
}

// AddSimpleSpec adds a simple spec to the swagger.
func AddSimpleSpec(swagger *Swagger, spec httpapi.SimpleSpec) {
	/*
		desc := spec.Descriptor()
		urlpath := desc.URLPath
		method := desc.Method
	*/
	panic("not implemented")
}

// AddTypedSpec adds a typed spec to the swagger.
func AddTypedSpec[T any](swagger *Swagger, spec httpapi.TypedSpec[T]) error {
	/*
		desc, err := spec.Descriptor()
		if err != nil {
			return err
		}
	*/
	panic("not implemented")
}
