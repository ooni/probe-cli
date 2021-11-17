// Package ooapi contains a client for the OONI API. We
// automatically generate the code in this package from the
// apimodel and internal/generator packages.
//
// Usage
//
// You need to create a Client. Make sure you set all
// the mandatory fields. You will then have a function
// for every supported OONI API. This function will
// take in input a context and a request. You need to
// fill the request, of course. The return value is
// either a response or an error.
//
// If an API requires login, we will automatically
// perform the login. If an API uses caching, we will
// automatically use the cache.
//
// Design
//
// Most of the code in this package is auto-generated from the
// data model in ./apimodel and the definition of APIs provided
// by ./internal/generator/spec.go.
//
// We keep the generated files up-to-date by running
//
//     go generate ./...
//
// We have tests that ensure that the definition of the API
// used here is reasonably close to the server's one.
//
// Testing
//
// The following command
//
//     go test ./...
//
// will, among other things, ensure that the our API spec
// is consistent with the server's one. Running
//
//     go test -short ./...
//
// will exclude most (slow) integration tests.
//
// Architecture
//
// The ./apimodel sub-package contains the definition of request
// and response messages. We rely on tagging to specify how
// we should encode and decode messages.
//
// The ./internal/generator sub-package contains code to generate most
// code in this package. In particular, the spec.go file is
// the specification of the APIs.
package ooapi
