// -=-=- StartHere -=-=-
//
// # Chapter XIII: Rewriting Web Connectivity
//
// This chapter contains an exercise. We are going to
// use the `measurex` API to rewrite part of the
// Web Connectivity network experiment.
// (This is probably the right place to prod you
// to go to the [ooni/spec](https://github.com/ooni/spec)
// repository, locate the ts-017-web-connectivity.md
// spec, and read it.)
//
// Read the spec? Good, so
// what we are more precisely going to do here
// is implement the network measurement part of
// Web Connectivity where we:
//
// 1. enumerate all the IP addresses of the target
// URL using the system resolver;
//
// 2. build endpoints with such IPs with a suitable
// port, thus obtaining a list of HTTP endpoints;
//
// 3. TCP connect each of the endpoints and save the
// results into a measurement object compatible
// with Web Connectivity's data format;
//
// 4. TLS handshake each endpoint (only if this
// makes sense, of course);
//
// 5. HTTP GET the URL and follow redirects until
// we reach a webpage, fetch the body, and store it
// for later analysis (which we'll not implement
// as part of this exercise).
//
// Let us now provide extra context that should
// help you figure out how to solve this exercise.
//
// ## Regarding points 3-4
//
// You already know all the primitives.
//
// ## Regarding point 5
//
// Historically this point has always been
// performed by a separate HTTP client. This
// means that any implementation:
//
// - will not include any TCP or TLS event
// generated during point 5 in the measurement;
//
// - most likely will resolve the URL's domain
// again (even though the probe-cli implementation
// uses a fake Resolver to avoid that);
//
// - tries every available IP address and stops
// at the first one to which it can connect to (which
// is what a naive HTTP client does, whereas a more
// advanced one likely tries a couple of addrs in
// parallel, especially when both IPv4 and IPv6
// are supported - this is also known as happy eyeballs).
//
// In terms of `measurex`, the best API to do what
// you're required to do in point 5 is probably
// `NewTracingHTTPTransportWithDefaultSettings`, which
// allows you to trace only the HTTP round trip and
// ignores any other event.
//
// Once you have such a transport, the best `Measurer`
// API for the task is probably `HTTPClientGET`.
//
// ## Other remarks
//
// You also need to learn about how to measure
// events at low level, which entails creating an
// instance of `MeasurementDB`, passing it to
// the relevant networking code, and then calling
// its `AsMeasurement` method to get back a
// measurement. (You can probably get an idea
// of how this is done in general by checking the
// implementation of `Measurer.TCPConnect`.)
//
// Hopefully, this should be enough information
// to help you tackle this task. As you see
// below, the main function is there empty waiting
// for your implementation. We will provide our
// own solution to this problem in the next chapter.
//
// (This file is auto-generated. Do not edit it directly! To apply
// changes you need to modify `./internal/tutorial/measurex/chapter13/main.go`.)
//
// ## The main.go file
//
// ```Go
package main

func main() {
}
