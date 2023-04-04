// -=-=- StartHere -=-=-
//
// # Chapter 1: Introduction and General Principle
//
// Connections and requests using common Internet protocols are made up by a set of subsequent operations (steps).
// Here are some examples:
//
// a) To connect to a QUIC endpoint, we do 2 subsequent steps:
// * DNS lookup, and
// * QUIC handshake.
//
// b) In order to do an HTTPS transaction we do 4 subsequent steps:
// * DNS lookup,
// * TCP three-way-handshake,
// * TLS handshake, and
// * HTTP transaction containing HTTP requests and responses.
//
// Most OONI experiments observe and interpret the events during these operations. Thus, it makes sense to write experiments in a step-by-step manner as well, by building network functions from a toolbox of smaller building blocks.
//
// dslx provides such a toolbox of building blocks, in particular:
// * DNSLookupGetaddrinfo
// * DNSLookupUDP
// * TCPConnect
// * TLSHandshake
// * QUICHandshake
// * HTTPRequestOverTCP (HTTP)
// * HTTPRequestOverTLS (HTTPS)
// * HTTPRequestOverQUIC (HTTP/3)
//
// We can run a building block individually, e.g. the DNS Resolve operation:
//
// ```golang
// // pseudo code
// fn := dslx.DNSLookupGetaddrinfo()
// dnsResult := fn.Apply()
// ```
//
// We can put building blocks together, by using function composition:
//
// a)
// ```golang
// // pseudo code
// pipeline := dslx.Compose2(
//    DNSLookupGetaddrinfo(),
//    QUICHandshake(),
// )
// // Apply tries to execute both steps,
// // the DNS Lookup and the QUIC handshake.
// // If the DNS Lookup fails, the QUIC handshake is skipped.
// totalResult := pipeline.Apply()
// ```
//
// b)
// ```golang
// // pseudo code
// pipeline := dslx.Compose4(
//    DNSLookupGetaddrinfo(),
//    TCPConnect(),
//    TLSHandshake(),
//    HTTPRequestOverTLS(),
// )
// // Apply tries to execute all 4 steps: the DNS Lookup, TCP Connect,
// // TLS handshake and HTTP transaction.
// // If one step fails, the subsequent steps are skipped.
// totalResult := pipeline.Apply()
// ```

// ```Go
package main

// ```
