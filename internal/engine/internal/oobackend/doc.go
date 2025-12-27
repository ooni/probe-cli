// Package oobackend communicates with the OONI backend.
//
// Usage
//
// You need to create an instance of Client. Make sure you fill
// the mandatory fields. Then, you can pass Client as an http.Client-like
// data structure to other pieces of code that need it.
//
// The Client.Do method MAY copy requests and operate on such copies. It
// MAY, for example, rewrite the URL of the copied request, add or remove
// HTTP headers, etc.
//
// Behavior
//
// The Client will dynamically choose the best strategy for communicating
// with the OONI backends. Periodically, the Client will reset so to
// re-evaluate all the available strategies.
//
// Hovever, If you configure a specific proxy, only such a proxy will be
// used. We won't use dynamic strategies.
package oobackend
