
# Chapter I: using the system resolver

In this chapter we explain how to measure DNS resolutions performed
using the system resolver. *En passant*, we will also introduce you to
the `Measurer`, which we will use for the rest of the tutorial.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter01/main.go`.)

## The system resolver

We define "system resolver" as the DNS resolver implemented by the C
library. On Unix, the most popular interface to such a resolver is
the `getaddrinfo(3)` C library function.

Most OONI experiments (also known as nettests) use the system
resolver to map domain names to IP addresses. The advantage of
the system resolver is that it's provided by the system. So,
it should _generally_ work. Also, it is the resolver that the
user of the system will use every day, therefore its results
should be representative (even though the rise of DNS over
HTTPS embedded in browsers may make this statement less solid
than it were ten years ago).

The disadvantage of the system resolver is that we do not
know how it is configured. Say the user has configured a
DNS over TLS resolver; then the measurements may miss censorship
that we would otherwise see if using a custom DNS resolver.

Now that we have justified why the system resolver is
important for OONI, let us perform some measurements with it.

We will first write a simple `main.go` file that shows how to use
this functionality. Then, we will show some runs of this file, and
we will comment the output that we see.

## main.go

We declare the package and import useful packages. The most
important package we're importing here is, of course, `internal/measurex`.

```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
```
### Setup

We define command line flags useful to test this program. We use
the `flags` package for that. We want the user to be able to configure
both the domain name to resolve and the resolution timeout.

```Go
	domain := flag.String("domain", "example.com", "domain to resolve")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
```

We call `flag.Parse` to parse the CLI flags.

```Go
	flag.Parse()
```

We create a context and we attach a timeout to it. (This is a pretty
standard way of configuring a timeout in Go.)

```Go
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

### Creating a Measurer

Now we create a `Measurer`.

```Go
	mx := measurex.NewMeasurerWithDefaultSettings()
```

The `Measurer` is a concrete type that contains many fields
requiring initialization. For this reason, we provide a factory
that creates one with default settings. The expected usage
pattern is that you do not modify a `Measurer`'s field after
initialization. Modifying them while the `Measurer` is in
use could, in fact, lead to races.

Let's now invoke the system resolver to resolve `*domain`!

### Invoking the system resolver

We call the `LookupHostSystem` method of the `Measurer`. The
arguments are the Context, that in this case carries the timeout
we configured above, and the domain to resolve.

The call itself is named `LookupHost` because this is the name
used by the Go function that performs a domain lookup.

Under the hood, `mx.LookupHostSystem` will eventually call
`(*net.Resolver).LookupHost`. In turn, in the common case on
Unix, this function will eventually call `getaddrinfo(3)`.

```Go
	m := mx.LookupHostSystem(ctx, *domain)
```

The return value of `(*net.Resolver).LookupHost` is either a
list of IP addresses or an error. Our `LookupHostSystem` method,
instead, returns a `*measurex.DNSMeasurement` type.

This is probably a good moment to remind you of Go's
built in help system. We could include a definition of the
`DNSMeasurement` structure, but since this definition is
just a comment in the main.go file, it might age badly.

Instead, if you run

```
go doc ./internal/measurex.DNSMeasurement
```

You get the current definition. As you can see, this type
is basically just a wrapper around `Measurement`. Now,
checking the docs of `Measurement` with

```
go doc ./internal/measurex.Measurement
```

we can see a container of events
classified by event type. In our case, because we're
doing a `LookupHost`, we should have at least one entry
inside of the `Measurement.LookupHost` field.

This entry is of type `DNSLookupEvent`. Let us check
together the definition of this type:

```
go doc ./internal/measurex.DNSLookupEvent
```

If you are familiar with [the OONI data format specs](
https://github.com/ooni/spec/tree/master/data-formats), you
should probably recognize that this structure is the Go
representation of the `df-002-dnst` data format.

In fact, every event field inside of a `Measurement`
should serialize nicely to JSON to one of the OONI data
formats.

### Printing the measurement

Because there is a close relationship between the
events inside a `Measurement` and the JSON OONI data
format, in the remainder of this program we're
going to serialize the `Measurement` to JSON and
print it to the standard output.

Rather than serializing the raw `Measurement` struct,
we first convert it to the "archival" format. This is the
data format specified at [ooni/spec](https://github.com/ooni/spec/tree/master/data-formats).

```Go
	data, err := json.Marshal(measurex.NewArchivalDNSMeasurement(m))
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
```

As a final note, the `PanicOnError` is here because the
message `m` *can* be marshalled to JSON. It still feels a
bit better having an assertion for our assumptions than
outrightly ignoring the error code. (We tend to use such
a convention quite frequently in the OONI codebase.)

```Go
}

```

## Running the example program

Let us run the program with default arguments first. You can do
this operation by running:

```bash
go run -race ./internal/tutorial/measurex/chapter01 | jq
```

Where `jq` is being used to make the output more presentable.

If you do that you obtain some logging messages, which are out of
the scope of this tutorial, and the following JSON:

```JSON
{
  "domain": "example.com",
  "queries": [
    {
      "answers": [
        {
          "answer_type": "A",
          "ipv4": "93.184.216.34"
        }
      ],
      "engine": "system",
      "failure": null,
      "hostname": "example.com",
      "query_type": "A",
      "resolver_address": "",
      "t": 0.002996459,
      "started": 9.8e-05,
      "oddity": ""
    },
    {
      "answers": [
        {
          "answer_type": "AAAA",
          "ivp6": "2606:2800:220:1:248:1893:25c8:1946"
        }
      ],
      "engine": "system",
      "failure": null,
      "hostname": "example.com",
      "query_type": "AAAA",
      "resolver_address": "",
      "t": 0.002996459,
      "started": 9.8e-05,
      "oddity": ""
    }
  ]
}
```

This JSON [implements the df-002-dnst](https://github.com/ooni/spec/blob/master/data-formats/df-002-dnst.md)
OONI data format.

You see that we have two messages here. OONI splits a DNS
resolution performed using the system resolver into two "fake"
DNS resolutions for A and AAAA. (Under the hood, this is
what the system resolver would most likely do.)

The most important fields are:

- _engine_, indicating that we are using the "system" resolver;

- _hostname_, meaning that we wanted to resolve the "example.com" domain;

- _answers_, which contains a list of answers;

- _t_, which is the time when the LookupHost operation completed.

### NXDOMAIN measurement

Let us now change the domain to resolve to be `antani.ooni.org` (a
nonexisting domain), which we can do by running this command:

```bash
go run -race ./internal/tutorial/measurex/chapter01 -domain antani.ooni.org | jq
```

This is the output JSON:

```JSON
{
  "domain": "antani.ooni.org",
  "queries": [
    {
      "answers": null,
      "engine": "system",
      "failure": "dns_nxdomain_error",
      "hostname": "antani.ooni.org",
      "query_type": "A",
      "resolver_address": "",
      "t": 0.072963834,
      "started": 0.000125417,
      "oddity": "dns.lookup.nxdomain"
    },
    {
      "answers": null,
      "engine": "system",
      "failure": "dns_nxdomain_error",
      "hostname": "antani.ooni.org",
      "query_type": "AAAA",
      "resolver_address": "",
      "t": 0.072963834,
      "started": 0.000125417,
      "oddity": "dns.lookup.nxdomain"
    }
  ]
}
```

So we see a failure that says there was indeed an NXDOMAIN
error and we also see a field named `oddity`.

What is an oddity? We define oddity something unexpected thay
may be explained by censorship as well as by a transient failure
or other normal network conditions. (In this case, the result
is perfectly normal since we're looking up a nonexistent domain.)

The difference between failure and oddity is that the failure
indicates the error that occurred, while the oddity classifies
the error in the context of the operation during which it
occurred. (In this case the difference is subtle, but we'll
have a better example later, when we'll see what happens on timeout.)

Failures are specified in
[df-007-errors](https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md).
Inside the `internal/netxlite/errorsx`
package, there is code that maps Go errors to failures. (The
`netxlite` package is the fundamental network package we use, on
top of which `measurex` is written.)

### Measurement with timeout

Let us now try with an insanely low timeout:

```bash
go run -race ./internal/tutorial/measurex/chapter01 -timeout 250us | jq
```

To get this JSON:

```JSON
{
  "domain": "example.com",
  "queries": [
    {
      "answers": null,
      "engine": "system",
      "failure": "generic_timeout_error",
      "hostname": "example.com",
      "query_type": "A",
      "resolver_address": "",
      "t": 0.000489167,
      "started": 9.2583e-05,
      "oddity": "dns.lookup.timeout"
    },
    {
      "answers": null,
      "engine": "system",
      "failure": "generic_timeout_error",
      "hostname": "example.com",
      "query_type": "AAAA",
      "resolver_address": "",
      "t": 0.000489167,
      "started": 9.2583e-05,
      "oddity": "dns.lookup.timeout"
    }
  ]
}
```

You should now better see the difference between a failure and
an oddity. The context timeout maps to a `generic_timeout_error` while
the oddity clearly indicates the timeout happens during a DNS
lookup. As we mentioned above, the failure is just an error while
an oddity is an error put in context.

## Conclusions

This is it. We have seen how to measure with the system resolver and we have
also seen which easy-to-provoke errors we can get.

