# OONI Network Extensions

| Author       | Simone Basso |
|--------------|--------------|
| Last-Updated | 2020-04-02   |
| Status       | approved     |

## Introduction

OONI experiments send and/or receive network traffic to
determine if there is blocking. We want the implementation
of OONI experiments to be as simple as possible. We also
_want to attribute errors to the major network or protocol
operation that caused them_.

At the same time, _we want an experiment to collect as much
low-level data as possible_. For example, we want to know
whether and when the TLS handshake completed; what certificates
were provided by the server; what TLS version was selected;
and so forth. These bits of information are very useful
to analyze a measurement and better classify it.

We also want to _automatically or manually run follow-up
measurements where we change some configuration properties
and repeat the measurement_. For example, we may want to
configure DNS over HTTPS (DoH) and then attempt to
fetch again an URL. Or we may want to detect whether
there is SNI bases blocking. This package allows us to
do that in other parts of probe-engine.

## Rationale

As we observed [ooni/probe-engine#13](
https://github.com/ooni/probe-engine/issues/13), every
experiment consists of two separate phases:

1. measurement gathering

2. measurement analysis

During measurement gathering, we perform specific actions
that cause network data to be sent and/or received. During
measurement analysis, we process the measurement on the
device. For some experiments (e.g., Web Connectivity), this
second phase also entails contacting OONI backend services
that provide data useful to complete the analysis.

This package implements measurement gathering. The analysis
is performed by other packages in probe-engine. The core
design idea is to provide OONI-measurements-aware replacements
for Go standard library interfaces, e.g., the
`http.RoundTripper`. On top of that, we'll create all the
required interfaces to achive the measurement goals mentioned above.

We are of course writing test templates in `probe-engine`
anyway, because we need additional abstraction, but we can
take advantage of the fact that the API exposed by this package
is stable by definition, because it mimics the stdlib. Also,
for many experiments we can collect information pertaining
to TCP, DNS, TLS, and HTTP with a single call to `netx`.

This code used to live at `github.com/ooni/netx`. On 2020-03-02
we merged github.com/ooni/netx@4f8d645bce6466bb into `probe-engine`
because it was more practical and enabled easier refactoring.

## Definitions

Consistently with Go's terminology, we define
_HTTP round trip_ the process where we get a request
to send; we find a suitable connection for sending
it, or we create one; we send headers and
possibly body; and we receive response headers.

We also define _HTTP transaction_ the process starting
with an HTTP round trip and terminating by reading
the full response body.

We define _netx replacement_ a Go struct of interface that
has the same interface of a Go standard library object
but additionally performs measurements.

## Enhanced error handling

This library MUST wrap `error` such that:

1. we can classify all errors we care about; and

2. we can map them to major operations.

The `github.com/ooni/netx/modelx` MUST contain a wrapper for
Go `error` named `ErrWrapper` that is at least like:

```Go
type ErrWrapper struct {
    Failure    string // error classification
    Operation  string // operation that caused error
    WrappedErr error  // the original error
}

func (e *ErrWrapper) Error() string {
    return e.Failure
}
```

Where `Failure` is one of the errors we care about, i.e.:

- `connection_refused`: ECONNREFUSED
- `connection_reset`: ECONNRESET
- `dns_bogon_error`: detected bogon in DNS reply
- `dns_nxdomain_error`: NXDOMAIN in DNS reply
- `eof_error`: unexpected EOF on connection
- `generic_timeout_error`: some timer has expired
- `ssl_invalid_hostname`: certificate not valid for SNI
- `ssl_unknown_autority`: cannot find CA validating certificate
- `ssl_invalid_certificate`: e.g. certificate expired
- `unknown_failure <string>`: any other error

Note that we care about bogons in DNS replies because they are
often used to censor specific websites.

And where `Operation` is one of:

- `resolve`: domain name resolution
- `connect`: TCP connect
- `tls_handshake`: TLS handshake
- `http_round_trip`: reading/writing HTTP

The code in this library MUST wrap returned errors such
that we can cast back to `ErrWrapper` during the analysis
phase, using Go 1.13 `errors` library as follows:

```Go
var wrapper *modelx.ErrWrapper
if errors.As(err, &wrapper) == true {
    // Do something with the error
}
```

## Netx replacements

We want to provide netx replacements for the following
interfaces in the Go standard library:

1. `http.RoundTripper`

2. `http.Client`

3. `net.Dialer`

4. `net.Resolver`

Accordingly, we'll define the following interfaces in
the `github.com/ooni/probe-engine/netx/modelx` package:

```Go
type DNSResolver interface {
	LookupHost(ctx context.Context, hostname string) ([]string, error)
}

type Dialer interface {
	Dial(network, address string) (net.Conn, error)
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type TLSDialer interface {
	DialTLS(network, address string) (net.Conn, error)
	DialTLSContext(ctx context.Context, network, address string) (net.Conn, error)
}
```

We won't need an interface for `http.RoundTripper`
because it is already an interface, so we'll just use it.

Our replacements will implement these interfaces.

Using an API compatible with Go's standard libary makes
it possible to use, say, our `net.Dialer` replacement with
other libraries. Both `http.Transport` and
`gorilla/websocket`'s `websocket.Dialer` have 
functions like `Dial` and `DialContext` that can be
overriden. By overriding such function pointers,
we could use our replacements instead of the standard
libary, thus we could collect measurements while
using third party code to implement specific protocols.

Also, using interfaces allows us to combine code
quite easily. For example, a resolver that detects
bogons is easily implemented as a wrapper around
another resolve that performs the real resolution.

## Dispatching events

The `github.com/ooni/netx/modelx` package will define
an handler for low level events as:

```Go
type Handler interface {
    OnMeasurement(Measurement)
}
```

We will provide a mechanism to bind a specific
handler to a `context.Context` such that the handler
will receive all the measurements caused by code
using such context. This mechanism is like:

```Go
type MeasurementRoot struct {
	Beginning time.Time // the "zero" time
	Handler Handler     // the handler to use
}
```

You will be able to assign a `MeasurementRoot` to
a context by using the following function:

```Go
func WithMeasurementRoot(
    ctx context.Context, root *MeasurementRoot) context.Context
```

which will return a clone of the original context
that uses the `MeasurementRoot`. Pass this context to
any method of our replacements to get measurements.

Given such context, or a subcontext, you can get
back the original `MeasurementRoot` using:

```Go
func ContextMeasurementRoot(ctx context.Context) *MeasurementRoot
```

which will return the context `MeasurementRoot` or
`nil` if none is set into the context. This is how our
internal code gets access to the `MeasurementRoot`.

## Constructing and configuring replacements

The `github.com/ooni/probe-engine/netx` package MUST provide an API such
that you can construct and configure a `net.Resolver` replacement
as follows:

```Go
r, err := netx.NewResolverWithoutHandler(dnsNetwork, dnsAddress)
if err != nil {
    log.Fatal("cannot configure specifc resolver")
}
var resolver modelx.DNSResolver = r
// now use resolver ...
```

where `DNSNetwork` and `DNSAddress` configure the type
of the resolver as follows:

- when `DNSNetwork` is `""` or `"system"`, `DNSAddress` does
not matter and we use the system resolver

- when `DNSNetwork` is `"udp"`, `DNSAddress` is the address
or domain name, with optional port, of the DNS server
(e.g., `8.8.8.8:53`)

- when `DNSNetwork` is `"tcp"`, `DNSAddress` is the address
or domain name, with optional port, of the DNS server
(e.g., `8.8.8.8:53`)

- when `DNSNetwork` is `"dot"`, `DNSAddress` is the address
or domain name, with optional port, of the DNS server
(e.g., `8.8.8.8:853`)

- when `DNSNetwork` is `"doh"`, `DNSAddress` is the URL
of the DNS server (e.g. `https://cloudflare-dns.com/dns-query`)

When the resolve is not the system one, we'll also be able
to emit events when performing resolution. Otherwise, we'll
just emit the `DNSResolveDone` event defined below.

Any resolver returned by this function may be configured to return the
`dns_bogon_error` if any `LookupHost` lookup returns a bogon IP.

The package will also contain this function:

```Go
func ChainResolvers(
    primary, secondary modelx.DNSResolver) modelx.DNSResolver
```

where you can create a new resolver where `secondary` will be
invoked whenever `primary` fails. This functionality allows
us to be more resilient and bypass automatically certain types
of censorship, e.g., a resolver returning a bogon.

The `github.com/ooni/probe-engine/netx` package MUST also provide an API such
that you can construct and configure a `net.Dialer` replacement
as follows:

```Go
d := netx.NewDialerWithoutHandler()
d.SetResolver(resolver)
d.ForceSpecificSNI("www.kernel.org")
d.SetCABundle("/etc/ssl/cert.pem")
d.ForceSkipVerify()
var dialer modelx.Dialer = d
// now use dialer
```

where `SetResolver` allows you to change the resolver,
`ForceSpecificSNI` forces the TLS dials to use such SNI
instead of using the provided domain, `SetCABundle`
allows to set a specific CA bundle, and `ForceSkipVerify`
allows to disable certificate verification. All these funcs
MUST NOT be invoked once you're using the dialer.

The `github.com/ooni/probe-engine/netx` package MUST contain
code so that we can do:

```Go
t := netx.NewHTTPTransportWithProxyFunc(
    http.ProxyFromEnvironment,
)
t.SetResolver(resolver)
t.ForceSpecificSNI("www.kernel.org")
t.SetCABundle("/etc/ssl/cert.pem")
t.ForceSkipVerify()
var transport http.RoundTripper = t
// now use transport
```

where the functions have the same semantics as the
namesake functions described before and the same caveats.

We also have syntactic sugar on top of that and legacy
methods, but this fully describes the design.

## Structure of events

The `github.com/ooni/probe-engine/netx/modelx` will contain the
definition of low-level events. We are interested in
knowing the following:

1. the timing and result of each I/O operation.

2. the timing of HTTP events occurring during the
lifecycle of an HTTP request.

3. the timing and result of the TLS handshake including
the negotiated TLS version and other details such as
what certificates the server has provided.

4. DNS events, e.g. queries and replies, generated
as part of using DoT and DoH.

We will represent time as a `time.Duration` since the
beginning configured either in the context or when
constructing an object. The `modelx` package will also
define the `Measurement` event as follows:

```Go
type Measurement struct {
    Connect             *ConnectEvent
    HTTPConnectionReady *HTTPConnectionReadyEvent
    HTTPRoundTripDone   *HTTPRoundTripDoneEvent
    ResolveDone         *ResolveDoneEvent
    TLSHandshakeDone    *TLSHandshakeDoneEvent
}
```

The events above MUST always be present, but more
events will likely be available. The structure
will contain a pointer for every event that
we support. The events processing code will check
what pointer or pointers are not `nil` to known
which event or events have occurred.

To simplify joining events together the following holds:

1. when we're establishing a new connection there is a nonzero
`DialID` shared by `Connect` and `ResolveDone`

2. a new connection has a nonzero `ConnID` that is emitted
as part of a successful `Connect` event

3. during an HTTP transaction there is a nonzero `TransactionID`
shared by `HTTPConnectionReady` and `HTTPRoundTripDone`

4. if the TLS handshake is invoked by HTTP code it will have a
nonzero `TrasactionID` otherwise a nonzero `ConnID`

5. the `HTTPConnectionReady` will also see the `ConnID`

6. when a transaction starts dialing, it will pass its
`TransactionID` to `ResolveDone` and `Connect`

7. when we're dialing a connection for DoH, we pass the `DialID`
to the `HTTPConnectionReady` event as well

Because of the following rules, it should always be possible
to bind together events. Also, we define more events than the
above, but they are ancillary to the above events. Also, the
main reason why `HTTPConnectionReady` is here is because it is
the event allowing to bind `ConnID` and `TransactionID`.
