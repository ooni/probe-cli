// Package ooapi contains clients for the OONI API. We
// automatically generate the code in this package from
// the apimodel and internal/generator packages. For
// each OONI API, we define up to three data structures:
//
// 1. a data structure representing the API;
//
// 2. a caching data structure, if the API
// supports caching;
//
// 3. an auto-login data structure, if the API
// requires login.
//
// The rest of this documentation page describes these
// three data structures and the design and architecture
// of this package. Refer to subpackages for more
// information on how to specify an API.
//
// API data structure
//
// For each API, this package defines a data structure
// representing the API. For example, for the TorTargets API,
// we define the TorTargetsAPI data structure.
//
// The API data structure defines a method named Call that
// allows calling the specified API. Call takes as arguments
// a context and the request for the API and returns the
// API response or an error.
//
// Request and response messages live inside the apimodel
// subpackage. We name them after the API. Thus, for
// the TorTargets API, the request is TorTargetsRequest,
// and the response is TorTargetsResponse.
//
// API data structures are cheap to create and do not
// mutate. They should be used in place and then forgotten
// off once the API call is complete.
//
// Unless explicitly indicated, the zero value of every
// API data structure is a valid API data structure.
//
// In terms of dependencies, APIs certainly need an http.Client
// to communicate with the OONI backend. To represent such a
// client, we use the HTTPClient interface. If you do not tell
// an API which http.Client to use, we will default to the
// standard library's http.DefaultClient.
//
// An API also depends on a JSONCodec. That is, on a data
// structures that encodes data to/from JSON. If you do not
// specify explicitly a JSONCodec, we will use the Go
// standard library's JSON implementation.
//
// When an API requires authentication, you need to tell
// it which authentication token to use. This gives you
// control over obtaining the token and is the low-level
// way of interacting with authenticated APIs. We recommend
// using the auto-login wrappers instead (see below).
//
// Authenticated APIs also define the WithToken method. This
// method takes as argument a token and returns a copy of the
// original API using the given token. We use this method
// to implement auto-login wrappers.
//
// For each API, we also define two interfaces:
//
// 1. the Caller interface represents the possibility of
// calling a specific API with the correct arguments;
//
// 2. the Cloner interface represents the possibility of
// calling WithToken on the given API.
//
// They abstract the interaction between the API type and
// its caching and auto-login wrappers.
//
// Caching
//
// If an API supports caching, we define a type whose name
// ends in Cache. The TorTargets API cache, for example,
// is TorTargetsCache. These caching types wrap the API type
// and provide the caching functionality.
//
// Because the cache needs to read from and write to the
// disk, a caching type needs a KVStore. A KVStore is
// an interface that allow you to bind a specific key to
// a given blob of bytes and to retrieve such bytes later.
//
// Caches use the gob data format from the Go standard
// library (`encoding/gob`). We abstract this dependency
// using the GobCodec interface. By default, when you
// do not specify a GobCodec we use the implementation
// of gob from the Go standard library.
//
// See the example describing caching for more information
// on how to use caching.
//
// Auto-login
//
// If an API supports auto-login, we define a type whose
// name ends with WithLogin. The TorTargets auto-login struct,
// for example, is called TorTargetsAPIWithLogin.
//
// Auto-login wrappers need to store persistent data. We
// use a KVStore for that (see above). We encode login data
// using JSON. To this end, we use a JSONCodec (also
// described above).
//
// See the example describing auto-login for more information
// on how to use auto-login.
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
// The ./apimodel package contains the definition of request
// and response messages. We rely on tagging to specify how
// we should encode and decode messages.
//
// The ./internal/generator contains code to generate most
// code in this package. In particular, the spec.go file is
// the specification of the APIs.
//
// Notable generated files
//
// - apis.go: contains APIs (e.g., TorTargetsAPI);
//
// - caching.go: contains caching wrappers for every API
// that declares that it needs a cache (e.g., TorTargetsCache);
//
// - callers.go: contains Callers;
//
// - cloners.go: contains the Cloners;
//
// - login.go: contains auto-login wrappers (e.g.,
// TorTargetsAPIWithLogin);
//
// - requests.go: contains code to generate http.Requests.
//
// - responses.go: code to parse http.Responses.
package ooapi
