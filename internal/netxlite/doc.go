// Package netxlite contains network extensions.
//
// This package is the basic networking building block that you
// should be using every time you need networking.
//
// It implements interfaces defined in the internal/model package.
//
// You should consider checking the tutorial explaining how to use this package
// for network measurements: https://github.com/ooni/probe-cli/tree/master/internal/tutorial/netxlite.
//
// Naming and history
//
// Previous versions of this package were called netx. Compared to such
// versions this package is lightweight because it does not contain code
// to perform the measurements, hence its name.
//
// Design
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
// Operations
//
// This package implements the following operations:
//
// 1. establishing a TCP connection;
//
// 2. performing a domain name resolution with the "system" resolver
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
package netxlite
