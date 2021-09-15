# Tutorial: network measurements with internal/measure

This tutorial explains how to use the [internal/measure](../../measure)
Go package to perform network measurements.

## Introduction

The `internal/measure` package separates DNS resolutions code from
code that measures endpoints. (In this context, with endpoint we mean
a TCP/UDP endpoint composed of an IPv4/IPv6 address and a port.)

We separate DNS from endpoint measurement because we noticed cases where,
e.g., `8.8.4.4:443/tcp` was working and `8.8.8.8:443/tcp` was blocked. For
this reason, we want to measure all of a service's endpoints.

The `internal/measure` library implements these *operations*:

- resolving domain names using the system resolver;

- connecting to a TCP endpoint;

- performing a TLS handshake given a connection and a TLS configuration;

- performing a QUIC handshake given an UDP endpoint and a TLS configuration;

- sending a DNS query over a DNS-over-UDP, DNS-over-TCP, DNS-over-TLS,
or DNS-over-HTTPS transport and reading the corresponding response;

- sending a GET request using HTTP/HTTP2/HTTP3 over an already
established TCP/TLS/QUIC channel to an HTTP server and reading the
corresponding response.

Users of `internal/measure` create an instance of the `Measurer`
structure and call one of its methods to perform a specific
kind of measurement. Each method is a *flow* (i.e., a sequence
of one or more of the fundamental operations described above). The
`Measurer.HTTPSEndpointGet` function, for example, implements
this flow:

- connecting to a given TCP endpoint;

- performing a TLS handshake;

- sending an HTTP request and reading the response.

In this tutorial we will learn about the most frequently used
flows, we will describe the data structures representing the
results, and we will show how they change in presence of network
anomalies.

## Index

This tutorial is organized as follows:

In the [first chapter](chapter01) we introduce the `Measurer` and
we measure DNS resolutions using the system resolver.

In the [second chapter](chapter02) we measure the establishment of TCP connections.

In the [third chapter](chapter03) we measure DNS resolutions
using a DNS-over-UDP resolver.

In the [fourth chapter](chapter04) we measure the establishment of a
TLS session with a remote TCP endpoint.

In the [fifth chapter](chapter05) we measure the establishment
of a QUIC session with a remote UDP endpoint.

In the [sixth chapter](chapter06) we measure fetching a webpage over HTTPS.

In the [seventh chapter](chapter07) we start putting all the bits together
and we write a program that measures a single URL (without redirections).

Future chapters will cover these topics:

* following HTTP redirections

* measuring HTTP3 and HTTP endpoints

* converting `internal/measure` data format to the archival data
format currently being used by the OONI pipeline

## Regenerating this tutorial

Most of the text of these tutorials comes from comments in real
Go code, to ensure that the code we show is always working against
the main development branch. For this reason, one should not edit
the README.md files manually when a Go file is also present in the
same directory. The following command regenerates all tutorials.

```
(cd ./internal/tutorial && go run ./generator)
```