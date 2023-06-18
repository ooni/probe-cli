
# Chapter 1: Introduction and General Principle

Before implementing a complete OONI Probe experiment using `dslx` in the next chapter, we
will first learn about the basic principles behind the `dslx` API.

## Background: Step-by-step network operations

Connections and requests using common Internet protocols are made up by a set of subsequent
operations (steps). Here are some examples:

*Example A* To connect to a QUIC endpoint, we do 2 subsequent steps:

* DNS lookup, and
* QUIC handshake.

*Example B* In order to do an HTTPS transaction we do 4 subsequent steps:

* DNS lookup,
* TCP three-way-handshake,
* TLS handshake, and
* HTTP transaction containing HTTP requests and responses.

Most OONI experiments observe and interpret the events during these operations. Thus, it makes
sense to write experiments in a step-by-step manner as well, by building network functions
from a toolbox of smaller building blocks.

## `dslx` building blocks

`dslx` provides such a toolbox of building blocks, in particular:

* DNSLookupGetaddrinfo
* DNSLookupUDP
* TCPConnect
* TLSHandshake
* QUICHandshake
* HTTPRequestOverTCP (HTTP)
* HTTPRequestOverTLS (HTTPS)
* HTTPRequestOverQUIC (HTTP/3)

We can run a building block individually, e.g. the DNS lookup operation:

```golang
// pseudo code
fn := dslx.DNSLookupGetaddrinfo()
dnsResult := fn.Apply("ooni.org")
```

The first line creates `fn` as a lazy function allowing one to perform a DNS
lookup using getaddrinfo. The second line applies the lazy function to its
arguments and produces a results. We use lazy functions in `dslx` because that
allows us to compose lazy functions together to build pipelines, as we will
show in the next section.

The input of `fn` is the domain name to resolve (plus other options that we
do not mention here for brevity). The output of `fn` is the result of
performing a DNS lookup; i.e., either an error or a list of resolved IP addresses.

## `dslx` function composition

By using `dslx` function composition, we can put building blocks together to create
measurement pipelines. When calling `Apply` on such a pipeline, `dslx` tries to
execute all steps inside the pipeline. If one step fails, the subsequent steps are
skipped.

*Example A*

```Go
// pseudo code
pipeline := dslx.Compose2(
   DNSLookupGetaddrinfo(),
   QUICHandshake(),
)
totalResult := pipeline.Apply("ooni.org")
```

In this example we create a pipeline composes of two stages. The first stage performs
a DNS lookup using the `getaddrinfo` function. The output of this stage consists of
either IP addresses or an error. In case of error, the second stage will not run, as
mentioned previously. In case of success, the second stage will perform a QUIC
handshake using the IP addresses discovered in the previous stage plus some other
configuration parameters you would have provided to the constructor.

The generated pipeline takes in input the domain name to resolve (plus other options
not shown here for brevity). The output is the result of a QUIC handshake; i.e.,
either an error or a QUIC connection. In case the first stage fails, the code will
skip the QUIC handshake step and just emit the DNS lookup error.


*Example B*
```Go
// pseudo code
pipeline := dslx.Compose4(
   DNSLookupGetaddrinfo(),
   TCPConnect(),
   TLSHandshake(),
   HTTPRequestOverTLS(),
)
totalResult := pipeline.Apply("ooni.org")
```

This second example is similar to the previous one. The main difference is that here
we build a four stage pipeline. When this pipeline runs successfully for the `ooni.org`
input, the output is the result of issuing an HTTP request to the website; i.e.,
either an HTTP response (on success) or an error (otherwise). As before, when a step
fails, we skip all the subsequent steps to immediately return an error.

Now that we have learned about this central and basic working principle of `dslx`, let's
start writing some actual experiment code [in chapter02](../chapter02/README.md)!

