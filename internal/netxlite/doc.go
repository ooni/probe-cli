// Package netxlite contains network extensions.
//
// This package is the basic networking building block that you
// should be using every time you need networking.
//
// It implements interfaces defined in internal/model/netx.go.
//
// You should consider checking the tutorial explaining how to use this package
// for network measurements: https://github.com/ooni/probe-cli/tree/master/internal/tutorial/netxlite.
//
// # Naming and history
//
// Previous versions of this package were called netx. Compared to such
// versions this package is lightweight because it does not contain code
// to perform the measurements, hence its name.
//
// # Design
//
// We want to potentially be able to observe each low-level operation
// separately, even though this is not done by this package. This is
// the use case where we are performing measurements.
//
// We also want to be able to use this package in a more casual way
// without having to compose each operation separately. This, instead, is
// the use case where we're communicating with the OONI backend.
//
// We want to optionally provide detailed logging of every operation,
// thus users can use `-v` to obtain OONI logs.
//
// We also want to mock any underlying dependency for testing.
//
// We also want to map errors to OONI failures, which are described by
// https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md.
//
// We want to have reasonable watchdog timeouts for each operation.
//
// We also want lightweight support for tracing network events. To this end, we
// use context.WithValue and context.Value to inject, and retrieve, a model.Trace
// implementation OPTIONALLY configured by the user.
//
// See also the design document at docs/design/dd-003-step-by-step.md,
// which provides an overview of netxlite's main concerns.
//
// To implement integration testing, we support hijacking the core network
// primitives used by this package, that is:
//
// 1. connecting a new TCP/UDP connection;
//
// 2. creating listening UDP sockets;
//
// 3. resolving domain names with getaddrinfo.
//
// By overriding the TProxy variable, you can control these operations and route
// traffic to, e.g., a wireguard peer where you implement censorship.
//
// # Operations
//
// This package implements the following operations:
//
// 1. establishing a TCP connection;
//
// 2. performing a domain name resolution with the "stdlib" resolver
// (i.e., getaddrinfo on Unix) or custom DNS transports (e.g., DoT, DoH);
//
// 3. performing the TLS handshake;
//
// 4. performing the QUIC handshake;
//
// 5. dialing with TCP, TLS, and QUIC (where in this context dialing
// means combining domain name resolution and "connecting");
//
// 6. performing HTTP, HTTP2, and HTTP3 round trips.
//
// Operations 1, 2, 3, and 4 are used when we perform measurements,
// while 5 and 6 are mostly used when speaking with our backend.
//
// # Getaddrinfo usage
//
// When compiled with CGO_ENABLED=1, this package will link with libc
// and call getaddrinfo directly. While this design choice means we will
// need to maintain more code, it also allows us to save the correct
// getaddrinfo return value, which is hidden by the Go resolver. Also,
// this strategy allows us to deal with the Android EAI_NODATA implementation
// quirk (see https://github.com/ooni/probe/issues/2029).
//
// We currently use net.Resolver when CGO_ENABLED=0. A future version of
// netxlite MIGHT change this and use a custom UDP resolver in such a
// case, to avoid depending on the assumption that /etc/resolver.conf is
// present on the target system. See https://github.com/ooni/probe/issues/2118
// for more details regarding ongoing plans to bypass net.Resolver when
// CGO_ENABLED=0. (If you're reading this piece of documentation and notice
// it's not updated, please submit a pull request to update it :-).
package netxlite
