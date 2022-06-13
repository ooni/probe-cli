# Step-by-step measurements

| | |
|:-------------|:-------------|
| Author | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2022-06-13   |
| Reviewed-by | [@hellais](https://github.com/hellais) |
| Reviewed-by | [@DecFox](https://github.com/DecFox/) |
| Status       | approved     |
| Obsoletes | [dd-002-netx.md](dd-002-netx.md) |

*Abstract.* The original [netx design document](dd-002-netx.md) is now two
years old. Since we wrote such a document, we amended the overall design
several times. The four major design changes where:

1. saving rather than emitting
[ooni/probe-engine#359](https://github.com/ooni/probe-engine/issues/359)

2. switching to save measurements using the decorator
pattern [ooni/probe-engine#522](https://github.com/ooni/probe-engine/pull/522);

3. the netx "pivot" [ooni/probe-cli#396](https://github.com/ooni/probe-cli/pull/396);

4. measurex [ooni/probe-cli#528](https://github.com/ooni/probe-cli/pull/528).

In this (long) design document, we will revisit the original problem proposed by
[df-002-netx.md](df-002-netx.md), in light of what we changed and of what we learned from the
changes we applied. We will highlight the major pain points of the current
implementation, which are these the following:

1. that the measurement library API is significantly different to the Go stdlib
API, therefore violating the original `netx` design goal that writing a new
experiment means using slightly different constructors that deviate from the
standard library only to meet specific measurement goals we have;

2. that the decorator pattern leads to complexity in creating measurement types,
which in turn seems to be the cause of the previous issue;

3. that the decorator pattern does not allow us to precisely collect all the
data that matters for events such as TCP connect and DNS round trips using
a custom transport, thus suggesting that we should revisit our choice of using
decorators and revert back to some form of _constructor based injection_ to
inject a data type suitable for saving events.

In doing that, we will also propose an incremental plan for moving the tree
forward from [the current state](https://github.com/ooni/probe-cli/tree/1685ef75b5a6a0025a1fd671625b27ee989ef111)
to a state where complexity is moved from the measurement-support library to
the implementation of each individual network experiment.

## Index

TBD

## netxlite: the underlying network library

This section describes `netxlite`, the underlying network library, from an
historical perspective. We explain our data collection needs and what types
from the standard library we're using as patterns.

### Measurement Observations

Most OONI experiment need to observe and give meaning to these events:

1. DNSLookup

2. TCPConnect

3. TLSHandshake

4. QUICHandshake

5. HTTP GET

6. TCP/UDP Read

7. TCP/UDP Write

8. UDP ReadFrom

9. UDP WriteTo

Observing Read, Write, ReadFrom, and WriteTo is optional. However, these
observations provide [information useful beyond just discussing the
blocking of resources](https://ooni.org/post/2022-russia-blocks-amid-ru-ua-conflict/#twitter-throttled).

As part of its life cycle, an OONI experiment performs these operations
multiple times. We call *observation* the result of each of these
network operations.

For each observation we want to collect when the related operation
started and terminated.

We also want to collect input parameters and output results.

When we're using a custom DNS transport (e.g., DNS over HTTPS), we
should also collect the exchanged DNS messages (query and response). In
this scenario, we may also want to record the child events caused by a
DNS round trip (e.g., TCPConnect, TLSHandshake).

When we're using getaddrinfo, we should [call it directly and collect
its return code](https://github.com/ooni/probe/issues/2029).

When we measure HTTP, there are redirections. Each redirection may or
may not reuse an existing TCP or QUIC connection. Each redirection has
an HTTP request and response. (Redirections are more complex than it
seems because of cookies; not entering into detail but worth
mentioning.)

The [OONI data format](https://github.com/ooni/spec/tree/master/data-formats)
defines how we archive experiment results as a set of observations.
(Orthogonally, we may also want to improve the data format, but this is
not under discussion now.)

### Error Wrapping

The OONI data format also specifies [how we should represent
errors](https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md).
Go generates its own errors and we should *reduce* those errors to the
set of strings specified in the OONI data format. (Orthogonally, we may
also want to introduce new, more precise errors when possible.)

We should also attribute the error to the operation that failed. In
principle, this seems easy. Yet, depending on how we're performing
measurements, it is not. More details later when appropriate.

A semi-orthogonal aspect is that we would also like to include in
collected measurements the underlying raw syscall or library errors that
occurred. That would be, e.g., getaddrinfo's return code or the Rcode of
DNS response messages or the syscall error returned by a Read call.

### Go Stdlib

The Go standard library provides the following structs and interfaces
that we can use for measuring:

```Go
// package net

type Resolver struct {}

func (r *Resolver) LookupHost(ctx context.Context, domain string) ([]string, error)
```

The Resolver is \~equivalent to calling getaddrinfo. However, we cannot
observe the error returned by getaddrinfo and we do not have the
guarantee that we're actually calling getaddrinfo. (On Unix, in
particular, [we use the "netgo" resolver](https://github.com/ooni/probe/issues/2118), which
reads `/etc/resolv.conf`, when `CGO_ENABLED=0`.)

```Go
// package net

type Dialer struct {}

func (d *Dialer) DialContext(ctx context.Context, network, address string) (net.Conn, error)
```

The Dialer combines DNSLookup and TCPConnect when the address contains a
TCP/UDP endpoint in which the hostname is not an IP address (e.g.,
`dns.google:443`). To observe a TCPConnect we need to make sure that we're
passing an address argument containing an IP address (e.g., `8.8.8.8:443`)
otherwise the whole operation will be a DNS lookup plus one or more
TCP-connect attempts.

```Go
// package crypto/tls

type Conn struct {}

func Client(conn net.Conn, config *tls.Config) *Conn

func (c *Conn) HandshakeContext(ctx context.Context) error

func (c *Conn) ConnectionState() tls.ConnectionState
```

The above APIs in `crypto/tls` allow us to perform a TLS handshake and
observe its results. The `crypto/tls` library is quite limited and this
[caused TLS fingerprinting issues in the
past](https://ooni.org/post/making-ooni-probe-android-more-resilient/).
To overcome this issue we devised two solutions:
[ooni/go](https://github.com/ooni/go) and
[ooni/oocrypto](https://github.com/ooni/oocrypto) (which
is leaner, but still has
[some issues](https://github.com/ooni/probe/issues/2122)).

```Go
// package net/http

type Transport struct {
	DialContext func(ctx context.Context, network, address string) (net.Conn, error)

	DialTLSContext func (ctx context.Context, network, address string) (net.Conn, error)

	// ...
}

func (txp *Transport) RoundTrip(req *http.Request) (*http.Response, error)

func (txp *Transport) CloseIdleConnections()

type RoundTripper interface {
	RoundTrip(req *http.Request) (*http.Response, error)
}

type HTTPClient struct {
	Transport http.RoundTripper
}
```

These APIs in `net/http` allow us to create connections and observe HTTP
round trips. The stdlib assumes we're using crypto/tls for TLS
connections and fails to establish HTTP2 connections otherwise because
it cannot read the ALPN array. So we [forked
net/http](https://github.com/ooni/oocrypto) to use
alternative TLS libs.

We could say more here. But I am trying to be brief. Because of that, I
am glossing over HTTP3, which is not part of the standard library but is
implemented by
[lucas-clemente/quic-go](https://github.com/lucas-clemente/quic-go).
Apart from the stdlib and quic-go, the only other major network code
dependency is [miekg/dns](https://github.com/miekg/dns)
for custom DNS resolvers.

### Network Extensions

A reasonable idea is to try to use types as close as possible to the
standard library. By following this strategy, we can compose our code
with stdlib code. We've been doing this [since day
zero](df-002-netx.md).

We use the `netx` name to identify **net**work e**x**tensions in ooni/probe-cli.

What is great about using stdlib-like types is that we're using code
patterns that people already know.

In this document, we're not going to discuss how netx should be
internally implemented. What matters is that we have a way to mimic
stdlib-like types. See
[internal/model/netx.go](https://github.com/ooni/probe-cli/blob/v3.15.1/internal/model/netx.go)
for details on those types.

In fact, what remains to discuss in this document is how we use these netx types to
perform measurements. And, this seems more of a software engineering
problem than anything else.

Yet, before jumping right into this topic, I think it is worth
mentioning that netx should do the following:

1. implement logging (we want ooniprobe -v to provide useful debug
information);

2. implement error wrapping and failed-operation mapping (as defined
above);

3. implement reasonable watchdog timeouts for every operation (OONI
runs in weird networks where censorship may cause OONI to become
stuck; see, for example, [ooni/probe#1609](https://github.com/ooni/probe/issues/1609)).

All network connections we create in OONI (be them for measuring or for
communicating with support services) have these concerns. Measurement
code has additional responsibilities, such as collecting and
interpreting the network observations. Separation of concerns therefore
suggests that measurement code be implemented by other packages using
netxlite as a dependency.

(The "lite" in `netxlite` reflects the fact that it does not concern
itself with measurements unlike [the original netx](df-002-netx.md), which contained
both basic networking wrappers and network measurement code.)
