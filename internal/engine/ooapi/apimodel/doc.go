// Package apimodel describes the data types used by OONI's API.
//
// If you edit this package to integrate the data model, remember to
// run `go generate ./...`.
//
// We annotate fields with tagging. When a field should be sent
// over as JSON, use the usual `json` tag.
//
// When a field needs to be sent using the query string, use
// the `query` tag instead. We limit what can be sent using the
// query string to int64, string, and bool.
//
// The `path` tag indicates that the URL path contains a
// template. We will replace the value of this field with
// the template. Note that the template should use the
// Go name of the field (e.g. `{{ .ReportID }}`) as opposed
// to the name in the tag, which is only used when we
// generate the API Swagger.
//
// The `required` tag indicates required fields. A required
// field cannot be empty (for the Go definition of empty).
package apimodel
