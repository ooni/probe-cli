// Package httpapi contains code for calling HTTP APIs.
//
// We model HTTP APIs as follows:
//
// 1. [Endpoint] is an API endpoint (e.g., https://api.ooni.io);
//
// 2. [Descriptor] describes the specific API you want to use (e.g.,
// GET /api/v1/test-list/urls with JSON response body).
//
// Generally, you use [Call] to call the API identified by a [Descriptor]
// on the specified [Endpoint]. However, there are cases where you
// need more complex calling patterns. For example, with [SequenceCaller]
// you can invoke the same API [Descriptor] with multiple equivalent
// API [Endpoint]s until one of them succeeds or all fail.
package httpapi
