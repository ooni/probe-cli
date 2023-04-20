// -=-=- StartHere -=-=-

// # Chapter 1: Introduction and General Principle

// Before implementing a complete OONI Probe experiment using `dslx` in the next chapter, we will first learn about the basic principles behind the `dslx` API.

// ## Background: Step-by-step network operations
// Connections and requests using common Internet protocols are made up by a set of subsequent operations (steps).
// Here are some examples:

// *Example A* To connect to a QUIC endpoint, we do 2 subsequent steps:
// * DNS lookup, and
// * QUIC handshake.

// *Example B* In order to do an HTTPS transaction we do 4 subsequent steps:
// * DNS lookup,
// * TCP three-way-handshake,
// * TLS handshake, and
// * HTTP transaction containing HTTP requests and responses.

// Most OONI experiments observe and interpret the events during these operations. Thus, it makes sense to write experiments in a step-by-step manner as well, by building network functions from a toolbox of smaller building blocks.

// ## `dslx` building blocks

// dslx provides such a toolbox of building blocks, in particular:
// * DNSLookupGetaddrinfo
// * DNSLookupUDP
// * TCPConnect
// * TLSHandshake
// * QUICHandshake
// * HTTPRequestOverTCP (HTTP)
// * HTTPRequestOverTLS (HTTPS)
// * HTTPRequestOverQUIC (HTTP/3)

// We can run a building block individually, e.g. the DNS Resolve operation:

// ```golang
// // pseudo code
// fn := dslx.DNSLookupGetaddrinfo()
// dnsResult := fn.Apply()
// ```

// ## `dslx` function composition
// By using `dslx` function composition, we can put building blocks together to create measurement pipelines. When calling `Apply` on such a pipeline, `dslx` tries to execute all steps inside the pipeline. If one step fails, the subsequent steps are skipped.

// *Example A*
// ```Go
// // pseudo code
// pipeline := dslx.Compose2(
//    DNSLookupGetaddrinfo(),
//    QUICHandshake(),
// )
// totalResult := pipeline.Apply()
// ```

// *Example B*
// ```Go
// // pseudo code
// pipeline := dslx.Compose4(
//    DNSLookupGetaddrinfo(),
//    TCPConnect(),
//    TLSHandshake(),
//    HTTPRequestOverTLS(),
// )
// totalResult := pipeline.Apply()
// ```

// Now that we have learned about this central and basic working principle of `dslx`, let's start writing some actual experiment code! [Goto chapter02](../chapter02/README.md).

