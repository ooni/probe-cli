# Directory github.com/ooni/probe-cli/internal

This directory contains private Go packages.

## Useful commands

You can read the Go documentation of a package by using `go doc -all`.

For example:

```bash
go doc -all ./internal/netxlite
```

You can get a graph of the dependencies using [kisielk/godepgraph](https://github.com/kisielk/godepgraph).

For example:

```bash
godepgraph -s -novendor -p golang.org,gitlab.com ./internal/engine | dot -Tpng -o deps.png
```

You can further tweak which packages to exclude by appending
prefixes to the list passed to the `-p` flag.

## Tutorials

The [tutorial](tutorial) package contains tutorials on writing new experiments,
using measurements libraries, and networking code.

## Network extensions

This section briefly describes the overall design of the network
extensions (aka `netx`) inside `ooni/probe-cli`. In OONI, we have
two distinct but complementary needs:

1. speaking with our backends or accessing other services useful
to bootstrap OONI probe and perform measurements;

2. implementing network experiments.

We originally implemented these functionality into a separate
repository: [ooni/netx](https://github.com/ooni/netx). The
original [design document](https://github.com/ooni/netx/blob/master/DESIGN.md)
still provides a good overview of the problems we wanted to solve.

The general idea was to provide interfaces replacing standard library
objects that we could further wrap to perform network measurements without
deviating from the normal APIs expected by Go programmers. For example,

```Go
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}
```

is a generic dialer that could be a `&net.Dialer{}` but could also be a
*saving* dialer that saves the results of dial events. So, you could write
something like:

```Go
saver := &Saver{}
var dialer Dialer = NewDialer()
dialer = saver.WrapDialer(dialer)
conn, err := dialer.DialContext(ctx, network, address)
events := saver.ExtractEvents()
```

In short, with the original `netx` you could write measurement code
resembling ordinary Go code but you could also save network events
from which to derive whether there was censorship.

Since then, the architecture itself has evolved and `netx` has been
merged into `ooni/probe-engine` and later `ooni/probe-cli`. As of
2022-06-06, these are the fundamental `netx` packages:

- [model/netx.go](model/netx.go): contains the interfaces and structs
patterned after the Go standard library used by `netx`;

- [netxlite](netxlite): implements error wrapping (i.e., mapping
Go errors to OONI errors), enforces timeouts, and generally ensures
that we're using a stdlib-like network API that meet all our
constraints and requirements (e.g., logging);

- [bytecounter](bytecounter): provides support for counting the
number of bytes consumed by network interactions;

- [multierror](multierror): defines an `error` type that contains
a list of errors for representing the results of operations where
multiple sub-operations may fail (e.g., TCP connect fails for
all the IP addresses associated with a domain name);

- [tracex](tracex): support for collecting events during operations
such as TCP connect, QUIC handshake, HTTP round trip. Collecting
events allows us to analyze such events and determine whether there
was blocking. This measurement strategy is called tracing because
we wrap fundamental types (e.g., a dialer or an HTTP transport) to
save the result of each operation into a "list of events" type
called `Saver;

- [engine/netx](engine/netx): code surviving from the original `netx`
implementation that we're still using for measuring. Issue
[ooni/probe#2121](https://github.com/ooni/probe/issues/2121) describes
a slow refactoring process where we'll move code outside of `netx`
and inside `netxlite` or other packages. We are currently experimenting
with step-by-step measurements, an alternative measurement
approach where we break down operations in simpler building blocks. This
alternative approach may eventually make `netx` obsolete.
