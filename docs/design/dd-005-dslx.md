# DSL for writing OONI Experiments

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2023-01-23                                     |
| Reviewed-by  | [@kelmenhorst](https://github.com/kelmenhorst) |
| Status       | approved                                       |

## Problem statement

This proposal introduces a Domain-Specific Language (DSL) for writing
OONI experiments. To understand why we advocate for adding such a DSL, we
must first discuss available strategies for writing OONI experiments
and their shortcomings.

### urlgetter
In probe-cli, we traditionally write network experiments using the
`urlgetter` library. The following example shows how to perform a
TLS handshake measurement using such a library:

```Go
getter := urlgetter.Getter{
	Begin: time.Now(),
	Config: urlgetter.Config{
		TLSServerName: "dns.google",
	},
	Session: session,
	Target: "tlshandshake://8.8.8.8:443/",
}
result, _ := getter.Get(ctx)
runtimex.Assert(result != nil, "expected non-nil result")
if result.Failure != nil {
	/* The operation failed. */
}
```

Here, we declare what we want to measure by initializing
the `Getter`. Then, we measure by calling `Get`. If not `nil`,
the returned `result` contains the measured observations
(i.e., the data describing the results of fundamental
network operations such as TCP connect or TLS handshake).

A lovely property of the `urlgetter` code is that it
is compact and declarative. One specifies the measurement
to perform by choosing the proper URL scheme. For
example, `tlshandshake://` instructs the code to perform
a TLS handshake, and `dnslookup://` to perform a DNS lookup.

However, adding a new option to the `Getter` struct leads
to adding a new code path to `Getter.Get`. In turn, we must
write tests to ensure the new option works as intended in
isolation and when combined with other options. Because the
`Getter` contains all possible code paths and several
options influence specific functionality such as DNS and
TLS, writing and maintaining proper unit tests
for `urlgetter` is problematic.

Additionally, `urlgetter` based code does not easily allow
one to introduce follow-up measurements if specific operations
performed by `Getter.Get` fail. (While OONI does not currently
run follow-up measurements, we know we will want to do that
eventually; therefore, it makes sense also to discuss the
measurement library's impact on these kinds of measurements.) With
`urlgetter`, a way to introduce follow-up measurements
is by inspecting the returned `result` and executing the
appropriate follow-up actions accordingly. For example,
one could write:

```Go
result, _ := getter.Get(ctx)
runtimex.Assert(result != nil, "expected non-nil result")
if result.FailedOperation == "tls_handshake" {
	/* Run follow up experiment(s) by processing result */
}
```

The resulting code structure is problematic because the failed operation
and the follow-up measurements code are distant in the source tree: the
failed operation lives inside `urlgetter`, but follow-up measurements live
in a specific experiment package. Thus, a change in `urlgetter` may also
have a ripple effect that propagates onto the experiment packages
(possibly in subtle ways).  One could otherwise consider implementing
follow-up measurements using options; however, doing that would only
worsen the situation in terms of testing.

### measurex

Aware of `urlgetter` shortcomings, we tried an alternative approach
with `measurex`. We based its implementation on the insight that
there are two basic classes of sequences of operations: DNS lookups
and endpoint operations. This insight stems from the `dnscheck`
implementation, where we explicitly separated DNS operations from
endpoint operations to independently measure each available IP
address. Accordingly, `measurex` exposes as building blocks the
most common DNS and endpoint sequences of operations:

1. DNS lookup using the system resolver;
2. DNS lookup using a given resolver (UDP, DoT, DoH);
3. TCP connect;
4. TCP connect followed by TLS handshake;
5. QUIC handshake;
6. TCP connect followed by HTTP GET;
7. TCP connect, followed by TLS handshake, and HTTP GET;
8. QUIC handshake followed by HTTP GET.

With `measurex`, we can rewrite the previous TLS handshake
example as follows:

```Go
type EndpointMeasurement struct {
	Network EndpointNetwork
	Address string
	*Measurement
}

// Measurement is a container for observations
type Measurement struct {
	Connect      []*NetworkEvent
	TLSHandshake []*QUICTLSHandshakeEvent
	/* ... */
}

/* ... */

mx := measurex.NewMeasurerWithDefaultSettings()
m := mx.TLSConnectAndHandshake(ctx, "8.8.8.8:443", &tls.Config{
	ServerName: "dns.google",
})
```

Checking whether `m` contained a failure is, unfortunately,
not practical. Neither `Measurement` nor `EndpointMeasurement`
expose any method to assist with that. While this was not
an issue when working with the websteps-illustrated prototype,
we soon discovered other issues that required us to refactor
and improve `measurex`. To this end, we introduced new methods
for structs and moved state between structs until the exposed
API was satisfactory for implementing websteps. At that
point, we started wondering whether the `measurex` design
based on sequences of operations captured what was essential
to measuring or whether new requirements or experiments
would have led us to want to refactor this library again.

### measurexlite

Because exposing sequences of operations was not fundamental
enough, we exposed operations directly, introducing
the step-by-step design and the
`measurexlite` library. Where `measurex` provides a list of
eight sequences of operations, `measurexlite` provides the
building blocks to implement such sequences (e.g., DNS
lookup,  TCP connect, and TLS handshake). For instance, here
is the previous example rewritten using the `measurexlite` API:

```Go
tx := measurexlite.NewTrace(traceID, time.Now())
dialer := tx.NewDialerWithoutResolver(logger)
conn, err := dialer.DialContext(ctx, "tcp", "8.8.8.8:443")
saveTCPConnectResults(tx.TCPConnects())
if err != nil {
	/* Here, one can execute follow-up actions */
	return
}
defer conn.Close()

tconfig := &tls.Config{
	ServerName: "dns.google",
}
thx := tx.NewTLSHandshakerStdlib(logger)
tconn, _, err := thx.Handshake(ctx, conn, tconfig)
saveTLSHandshakeResults(tx.TLSHandshakes())
saveNetworkEvents(tx.NetworkEvents())
if err != nil {
	/* Here, one can execute follow-up actions */
	return
}
defer tconn.Close()
```

The code can hardly be less abstract than when using
`measurexlite`. In fact, for each API in our low-level
`netxlite` network extensions library, there is a
corresponding, measurement-aware API in the `measurexlite`
library. In turn, `netxlite` only adds a tiny
abstraction layer around the Go stdlib.

It is apparent that `measurexlite` provides more control
to the programmer. Given an existing codebase written
using `measurexlite`, adding follow-up experiments boils
down to finding the right `if` block where to add code. At
the same time, code using `measurexlite` is significantly
more verbose and repetitive than `urlgetter` or `measurex`:
comparatively, `measurexlite` code feels like assembly.

When we introduced `measurexlite`, we were aware of these
shortcomings. We initially proposed solving them by
autogenerating code, a solution we used to generate the
Web Connectivity LTE implementation. However, we did
not have enough consensus that code generation was
the solution to avoid writing and maintaining the
`measurexlite` boilerplate.

Having discussed `urlgetter`, `measurex`, and `measurexlite`, we
are now well-positioned to diagnose the underlying issue.

## Diagnosis

The `urlgetter` library hides the composition of fundamental
building blocks and options behind the `Getter.Get` API. Therefore,
`urlgetter` tests must test all the possible combinations
of composing building blocks and options.

The `measurex` library exposes sequences of operations, thus
reducing the overall complexity. However these sequences are not
fundamental to the problem of performing network measurements. There
is a constant tension towards refactoring `measurex` to better
adapt it to the experiment on which we are currently working.

The `measurexlite` library exposes the fundamental network
operations. However, its API is too low-level and verbose to
be accepted by external contributors. Thus, we need to figure out
how to abstract away all the unimportant details.

We will solve this problem by allowing for easy and abstract
function composition. These are our building blocks:

1. DNS lookup using getaddrinfo;
2. DNS lookup using an abstract transport;
3. TCP connect;
4. TLS handshake over a stream-like connection;
5. QUIC handshake;
6. HTTP round trip over a stream-like connection.

Let us forget for a second that
we are using Go and imagine a language where the pipe operator `|`
means point-free function composition. Equipped with this
abstraction, we could rewrite the TCP connect plus TLS
handshake example we have been using so far as follows:

```
px = TCPConnect | TLSHandshake
```

This example captures what is fundamental about our problem and
abstracts implementation details away: the `px` "pipeline"
performs a TCP connect, and if successful, it performs a TLS handshake.

We cannot write Go code as abstract as this pipeline. However, let
us apply these concepts to Go code to increase abstraction. We will
start by exploring ways to express functional composition.

## Expressing function composition in Go

We are going to use Golang 1.18+ generics. To keep the examples
lightweight, however, we will sometimes omit some type parameters
when the meaning of the code is obvious to a human reader.

We represent a ~pure generic function using the `Func` interface:

```Go
type Func[A, B any] interface {
	Apply(ctx context.Context, a A) B
}
```

A and B  are Golang 1.18+ generic type parameters, and the `Apply`
method applies the function to its arguments. In addition to an
`A` parameter, we have a context parameter because `measurexlite`
operations always take such an argument.

A generic function will always return a generic `B`. However, because
the operations we are modeling may fail, we will return a `B` value
wrapped by a `Maybe` type:

```Go
type Operation[A, B any] = Func[A, *Maybe[B]]
```

(We will not use `Operation` in practice because the code is
more explicit if we always explicitly write the `Maybe` return value.)

In turn, `Maybe` models the possibility of failure:

```Go
type Maybe[State any] struct {
	Error error
	State State
}
```

If the operation succeeds, `Error` will be `nil`, and `State`
will be meaningful. Otherwise, `Error` will be non `nil`, and the
developer should consider the `State` invalid.

(There are many possible alternative names for `Maybe`. Rust, for example,
uses `Result`. We chose to use "maybe" because the sentence one gets
when reading out loud the type signature is very expressive: "this operation
takes in input an A and maybe returns a B but only in case of success;
otherwise it is an error.")

Let us now define the TLS handshake operation constructor. The TLS
handshake is an operation that, given a TCP connection, returns
a TLS connection or an error. Putting together all that we have said
so far, we will model it in Go as follows:

```Go
func TLSHandshake() Func[*TCPConnection, *Maybe[*TLSConnection]] {
	return &tlsHandshakeFunc{}
}

type tlsHandshakeFunc struct{}

// Apply implements Func
func (f *tlsHandshakeFunc) Apply(ctx context.Context, state *TCPConnection) *Maybe[*TLSConnection] {
	/* ... */
}
```

The `TCPConnection` struct will contain a TCP connection plus additional
state required by subsequent compatible stages (e.g., TLS handshake). As a
first approximation, we can assume that it will be like this:

```Go
type TCPConnection struct {
	Conn   net.Conn
	Domain string
}
```

We will clarify later why we need to know the domain. For now, let
us model the `TLSConnection` as follows:

```Go
type TLSConnection struct {
	Conn netxlite.TLSConn
}
```

It is now time to sketch out TCP connect. It is an operation that, given a TCP
endpoint, returns a TCP connection or an error. Hence:

```Go
func TCPConnect() Func[*Endpoint, *Maybe[*TCPConnection]] {
	return &tcpConnectFunc{}
}

type tcpConnectFunc struct{}

// Apply implements Func
func (f *tcpConnectFunc) Apply(ctx context.Context, state *Endpoint) *Maybe[*TCPConnection] {
	/* ... */
}
```

In turn, the `Endpoint` should contain the following fields:

```Go
type Endpoint struct {
	Domain  string
	Address string
	Network string
}
```

We will soon see why we need a `Domain`. For now, let us put everything
together so that we can implement the TCP connect and the TLS
handshake `Apply` methods as follows:

```Go
func (f *tcpConnectFunc) Apply(ctx context.Context, state *Endpoint) *Maybe[*TCPConnection] {
	dialer := netxlite.NewDialerWithoutResolver(/* ... */)
	conn, err := dialer.DialContext(ctx, state.Network, state.Address)
	return &Maybe[*TCPConnection]{
		Error: err,
		State: &TCPConnection{
			Conn: conn,
			Domain: state.Domain,
		}
	}
}

func (f *tlsHandshakeFunc) Apply(ctx context.Context, state *TCPConnection) *Maybe[*TLSConnection] {
	handshaker := netxlite.NewTLSHandshakerStdlib(/* ... */)
	config := &tls.Config{
		ServerName: state.Domain,
	}
	conn, _, err := handshaker.Handshake(ctx, state.Conn, config)
	return &Maybe[*TLSHandshake]{
		Error: err,
		State: &TLSConnection{
			Conn: conn,
		}
	}
}
```

From this example, why we needed a `Domain` field should now be
apparent. We use it to propagate the correct SNI value.

Having defined our atoms, let us at last implement composition:

```Go
func Compose(f Func[A, *Maybe[B]], g Func[B, *Maybe[C]]) Func[A, *Maybe[C]] {
	return &composeFunc{f, g}
}

type composeFunc struct{
	f Func[A, *Maybe[B]]
	g Func[B, *Maybe[C]]
}

// Apply implements Func
func (fx *composeFunc) Apply(ctx context.Context, state A) *Maybe[B] {
	r1 := fx.f.Apply(ctx, state)
	if r1.Error != nil {
		return &Maybe[B]{
			Error: r1.Error,
			State: *new(B), // this is the zero value
		}
	}
	return fx.g.Apply(ctx, r1.State)
}
```

The main job of function composition is to avoid calling the
second function if the first function fails: in case of
failure, we manually construct a Maybe containing an invalid
state and an error.

With composition implemented, we can finally write:

```Go
px := Compose(TCPConnect(), TLSHandshake())
```

We now have an abstract pipeline written in Go. However,
function composition was only the beginning of our journey. Lets
us now investigate collecting network observations as
part of composition.

## Composition also collects observations

Let us start by defining a container for network observations:

```Go
type Observation struct {
	NetworkEvents []*model.ArchivalNetworkEvent
	Queries       []*model.ArchivalDNSLookupResult
	/* ... */
}
```

Our job in this section is to figure out a way to automatically
create a list of these observations produced by each operation.

Because we are going to use `measurexlite` as the underlying library, let us
also assume we have a function that, given a `measurexlite` trace, produces
a list of observations by invoking the proper trace extractor methods:

```Go
func extractObservations(tx *measurexlite.Trace) []*Observations {
	/* ... */
}
```

To collect observations during TCP connect, we can modify the
definition of `Endpoint` to include a trace:

```Go
type Endpoint struct {
	/* ... */
	Trace *measurexlite.Trace
}
```

(In the actual implementation, we would have the endpoint store the
arguments required to create a trace rather than a trace, but doing that
here would have overcomplicated this text.)

Because we will need to use the same trace during the TLS handshake,
let us also add a trace to `TCPConnection`:

```Go
type TCPConnection struct {
	/* ... */
	Trace *measurexlite.Trace
}
```

By propagating the trace, we can collect observations. However, we also
need a way to extract them from a trace and store them somewhere else. Because
we need to collect observations regardless of whether the operation
succeeds, the right place is the `Maybe` type:

```Go
type Maybe struct {
	/* ... */
	Observations []*Observation
}
```

In terms of changing data structures, these changes were all we needed. We
can now update the implementation of our `Apply` methods by following
the step-by-step cookbook: we replace any `netxlite` API with the equivalent
`measurexlite` API:

```Go
func (f *tcpConnectFunc) Apply(ctx context.Context, state *Endpoint) *Maybe[*TCPConnection] {
	tx := state.Trace
	dialer := tx.NewDialerWithoutResolver()
	/* ... */
	return &Maybe[*TCPConnection]{
		/* ... */
		State: &TCPConnection{
			/* ... */
			Trace: tx,
		},
		Observations: collectObservations(tx),
	}
}

func (f *tlsHandshakeFunc) Apply(ctx context.Context, state *TCPConnection) *Maybe[*TLSConnection] {
	tx := state.Trace
	handshaker := tx.NewTLSHandshakerStdlib()
	/* ... */
	return &Maybe[*TLSHandshake]{
		/* ... */
		Observations: collectObservations(tx),
	}
}
```

The changes are minimal. We use the propagated trace to create the proper
measurement-aware `measurexlite` API. After the fundamental operation
terminates, we `collectObservations` from the trace and store them
in the returned `Maybe`.

The final touch is updating `Compose` to merge observations:

```Go
// Apply implements Func
func (fx *composeFunc) Apply(ctx context.Context, state A) *Maybe[B] {
	r1 := fx.f.Apply(ctx, state)
	if r1.Error != nil {
		return &Maybe[B]{
			/* ... */
			Observations: r1.Observations,
		}
	}
	r2 := fx.g.Apply(ctx, r1.State)
	r2.Observations = append(r2.Observations, r1.Observations...)
	return r2
}
```

Having implemented these changes and assuming we have a function
named `saveObservations` allowing us to save observations into the
test keys, we can write:

```Go
px := Compose(TCPConnect(), TLSHandshake())
endpoint := &Endpoint{ /* ... */ }
res := px.Apply(ctx, endpoint)
saveObservations(tk, res.Observations)
if res.Error != nil {
	return
}
```

This code allows us to create a pipeline (`px`) composed of several
fundamental building blocks, to collect observations, and to make
decisions depending on whether the pipeline succeeded.
It is now time to ensure we close open connections.

## Automatically closing connections

Until now, we glossed over closing open connections. However, a pipeline
may open TCP and TLS connections. Let us now propose a mechanism to
close these connections.
We need to define a `ConnPool` type:

```Go
type ConnPool struct{}

func (p *ConnPool) MaybeTrack(c io.Closer)

func (p *ConnPool) Close() error
```

One can register connections with `MaybeTrack`. If the `c` closer
is `nil`, `MaybeTrack` does nothing. Otherwise, `MaybeTrack` registers
`c` such that `ConnPool.Close` will close `c`.

Now that we have a `ConnPool`, let us use it. We modify the `TCPConnect`
operation constructor to be:

```Go
func TCPConnect(pool *ConnPool) Func[*Endpoint, Maybe[*TCPConnect]] {
	return &tcpConnectFunc{pool: pool}
}
```

We modify the `Apply` method to be like this:

```Go
func (f *tcpConnectFunc) Apply(ctx context.Context, state *Endpoint) *Maybe[*TCPConnection] {
	/* ... */
	conn, err := dialer.DialContext(ctx, state.Network, state.Address)
	f.pool.MaybeTrack(conn)
	/* ... */
}

```

We also apply similar changes to `TLSHandshake`. With these changes,
we do not need to worry about closing connections as long as we
declare a `ConnPool` as follows:

```Go
pool := &ConnPool{}
defer pool.Close()
```

This code ensures that we close all the connections opened by a given `px`
pipeline once we leave the function scope. Solving this problem required adding
arguments to operation constructors; let us now focus on passing optional
arguments to such constructors.

## Passing optional arguments to operation constructors

In most cases, one does not need to customize the behavior of operators,
but we need to allow for exceptions. For example, there are OONI experiments
where we use a custom X.509 certificate pool rather than the default one.
To support this use case, let
us, therefore, explore passing an optional X.509 certificate pool to the
TLS handshake constructor.

To start, we define an option as a function that modifies the private type
implementing the TLS handshake operator:

```Go
type TLSHandshakeOption func(*tlsHandshakeFunc)
```

We then implement the specific option we need as follows:

```Go
func TLSHandshakeOptionRootCAs(pool *x509.Pool) TLSHandshakeOption {
	return func(fx *tlsHandshakeFunc) {
		fx.pool = pool
	}
}
```

We also add the `pool` field:

```Go
type tlsHandshakeFunc struct {
	pool *x509.Pool
}
```

Moreover, we extend the constructor to support options:

```Go
func TLSHandshake(opts ...TLSHandshakeOption) Func[*TCPConnection, *Maybe[*TLSHandshake]] {
	fx := &tlsHandshakeFunc{
		pool: netxlite.NewDefaultCertPool(),
	}
	for _, opt := range opts {
		opt(fx)
	}
	return fx
}
```

These changes allow the programmer to optionally configure an X.509 certificate
pool. Using the same strategy, we can implement any other option
for any other structure. Let us now discuss parallelism.

## Parallel operations

Both `urlgetter` and `measurex` support running operations in parallel. With
`urlgetter`, you use a `Multi` to run several `Getter` types over a list of
inputs with a given parallelism:

```Go
multi := &urlgetter.Multi{
	Parallelism: 3,
}
input := []urlgetter.MultiInput{{
	Config: urlgetter.Config{
		TLSServerName: "dns.google",
	},
	Target: "tlshandshake://8.8.8.8:443/",
}, {
	Config: urlgetter.Config{
		TLSServerName: "dns.google",
	},
	Target: "tlshandshake://8.8.4.4:443/",
}
for out := range multi.Run(ctx, input) {
	tk := out.TestKeys
	runtimex.Assert(tk != nil, "got nil test keys")
}
```

With the websteps-illustrated `measurex` implementation, the
`MeasureEndpoints` function is intrinsically parallel:

```Go
mx := measurex.NewMeasurerWithDefaultSettings()
mx.Options.EndpointParallelism = 3
inputs := []*measurex.EndpointPlan{{
	Domain: "dns.google",
	Network: "tcp",
	Address: "8.8.8.8:443",
	URL: &measurex.SimpleURL{
		Scheme: "tlshandshake",
		Host: "8.8.8.8:443",
		Path: "/",
	},
}, {
	Domain: "dns.google",
	Network: "tcp",
	Address: "8.8.4.4:443",
	URL: &measurex.SimpleURL{
		Scheme: "tlshandshake",
		Host: "8.8.4.4:443",
		Path: "/",
	},
}}
for out := range mx.MeasureEndpoints(ctx, inputs...) {
	// Each out is an `*EndpointMeasurement` value
}
```

Because both libraries implement parallelism, also the DSL API
must support parallelism. To this end, let us define the
`Map` function:

```Go
func Map(
	ctx context.Context,
	parallelism int,
	px Func[A, *Maybe[B]],
	inputs <-chan A,
) <-chan *Maybe[B] {
	/* ... */
}
```

This function runs `parallelism` goroutines where each goroutine applies
the `px` pipeline to an argument read from `inputs`. By convention,
`Map` expects `input` to be closed to signal EOF. Similarly, `Map`
will close the returned channel when done writing it.

(In the current prototype, this function's name is `MapAsync`; we
should delete the `Map` function and rename `MapAsync` to `Map`. The
prototype also defines a type name `Streamable` that wraps the
channel, but that seems unnecessary.)

We also need a convenience function, `StreamList`, that takes
in input a list and returns a channel:

```Go
func StreamList(values ...T) <-chan T
```

This function creates a background goroutine that streams the
content of the list onto the channel, then closes the channel.

(The `StreamList` function is named `Stream` in the prototype,
and we should rename it before merging into probe-cli.)

Thanks to `StreamList`, we can write a `Map` example as follows:

```Go
inputs := StreamList(&Endpoint{
	Domain: "dns.google",
	Address: "8.8.8.8:443",
	Network: "tcp",
}, &Endpoint{
	Domain: "dns.google",
	Address: "8.8.8.8:443",
	Network: "tcp",
})
pool := &ConnPool{}
defer pool.Close()
px := Compose(TCPConnect(pool), TLSHandshake(pool))
for out := range Map(ctx, 3, px, inputs) {
	// Each out is a `*Maybe[*TLSConnection]` value
}
```

We are now able to run measurements in parallel. When doing that,
it is often convenient to count the times we reached a specific
pipeline stage, which we will discuss in the next section.

## Counting events

Counting the number of events is frequently helpful for determining
whether an experiment succeeded or failed. For example, let us
assume we are measuring TLS handshakes and want to know whether
at least one succeeded. We can do that using a `Counter` generic type:

```Go
const parallelism = 3
attempted := NewCounter[*Endpoint]()
tcpSuccess := NewCounter[*TCPConnection]()
tlsSuccess := NewCounter[*TLSConnection]()
px := Compose(
	attempted.Func(),
	TCPConnect(),
	tcpSuccess.Func(),
	TLSHandshake(),
	tlsSuccess.Func(),
)
for range Map(ctx, parallelism, px, inputs) {
	/* Drain the channel */
}
```

The code above increments a counter each time it reaches the
corresponding stage. If `inputs` contains three entries,
the `attempted` value will be three after we have drained the
channel. If a TCP connect fails and two succeed, the
`tcpSuccess` counter value will be two. If one TLS handshake
fails and the other succeeds, the `tlsSuccess` value will be one.

The `Counter` implementation is as follows:

```Go
type Counter[T any] struct {
	/* ... */
}

func NewCounter[T any]() *Counter {
	/* ... */
}

func (c *Counter) Func() Func[T, *Maybe[T]] {
	/* ... */
}

func (c *Counter) Value() int64 {
	/* ... */
}
```

(Note that this type has a different name in the prototype; we
should change its naming in the pull request to use the name used by this document.)

We have now shown how to count events. Another relatively frequent
measurement need is running a follow-up experiment when a specific pipeline stage fails.

## Running follow-up experiments

A follow-up experiment is an experiment started when, in another
experiment, a specific network operation fails. A typical example
is the following: we want to run a follow-up SNI blocking
measurement each time a TLS handshake fails with a connection
reset by peer error.

Neither `urlgetter` nor `measurex` directly supported running
follow-up measurements, as previously discussed. With the DSL API,
the most straightforward approach to writing follow-up experiments
consists of writing shorter pipelines. Let us assume, for the
sake of argument, that we have this pipeline:

```Go
px := Compose(
	TCPConnect(),
	TLSHandshake(),
	HTTPRequestOverTLS(),
)
result := px.Apply(ctx, endpoint)
saveObservations(tk, result.Observations)
```

Assuming we want to run an SNI blocking measurement in case of
TLS handshake failure, we can rewrite the code as follows:

```Go
px := Compose(TCPConnect(), TLSHandshake())
res := px.Apply(ctx, endpoint)
saveObservations(tk, res.Observations)
if res.Error != nil && res.Error.Error() == "connection_reset" {
	sniBlockingMeasurement(/* ..., */ res)
	return
}
httpPx := HTTPRequestOverTLS()
httpRes := httpPx.Apply(ctx, res.State)
saveObservations(tk, httpRes.Observations)
```

While fancier implementations are possible, the one above is a
very easy-to-implement and read algorithm, which correctly triggers
a follow-up measurement after a connection reset error. Also,
this implementation is such that the failed step and the follow-up
experiment are very close, increasing maintainability.

## Concurrency patterns

Let us conclude our analysis of the new proposed API by discussing
a typical concurrency pattern that occurs when performing network measurements.
Say we have a list of HTTPS endpoints to measure using TCP connect
and TLS handshake, and we want to issue the HTTP request just for
the first TLS connection that succeeds. We can structure our code as follows:

```Go
func measure(ctx context.Context, epnts ...*Endpoint) {
	px := Compose(
		TCPConnect(),
		TLSHandshake(),
	)
	const parallelism = 3
	ch := Map(ctx, parallelism, px, StreamList(epnts))
	found := false
	for  res := range ch {
		saveObservations(tk, res.Observations)
		if res.Error != nil || found {
			continue
		}
		found = true
		obs := runHTTPMeasurement(res)
		saveObservations(tk, obs)
	}
}
```

This code runs TCP connect and TLS handshake measurements
with parallelism three. Then we loop over the results
and only run HTTP measurements for the first successful result (if any).

Again, we could implement this functionality by adding extra
complexity to the DSL, but there is no need. (Still, the
prototype includes a feature allowing us to stop the pipeline
early, which is, in fact, extra complexity, and we should remove it.)

This section completed our design space exploration. Let us
now conclude by comparing the DSL to other APIs.

## Evaluation

The DSL API is more abstract and less verbose than the
`measurexlite` API. Introducing new functionality does not
cause too much churn in tests because we do not use a
single Config struct as in `urlgetter`. Thus, we do not
need to worry about the effect of an option on each other
option. Still, every pipeline stage needs to provide
subsequent stages with the required state variables,
entangling the stages. However, the amount
of entanglement is lower than if we had a single Config
struct as we do in `urlgetter`.

Adding new functionality to the DSL API should not
cause us to want to refactor the library because the
functions implementing the operations are very
close to pure functions. Most of the state lives
in separate structures; therefore, the code
that composes functions together is terse and should
not change. Additionally, we are dealing with
fundamental operations. Nevertheless, what could
change is the content of the structures containing
the state (e.g., `Endpoint`). The primary source of concern is
adding new state variables and forgetting about
updating all the places in which we initialize
a structure, thus leaving some fields zeroed.

Like the `urlgetter` API and the `measurex` API, this
new API supports performing parallel operations. All three
APIs return results onto a channel closed to signal
EOF. Unlike other APIs, the DSL API parallel operation,
called `Map`, also takes in input a channel, thus
allowing for more scalable operations.

Because function composition and function application
are two separate operations, we can easily interrupt
pipelines midway to switch from parallel to serial
operations and perform follow-up measurements that would
require writing more code and more tests with the
`urlgetter` or `measurex` APIs.
