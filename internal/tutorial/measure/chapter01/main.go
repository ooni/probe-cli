// -=-=- StartHere -=-=-
//
// # Chapter I: using the system resolver
//
// In this chapter we explain how to measure DNS resolutions performed
// using the system resolver. *En passant*, we will also introduce you to
// the `Measurer`, which we will use for the rest of the tutorial.
//
// We will first write a simple `main.go` file that shows how to use
// this functionality. Then, we will show some runs of this file, and
// we will comment the output that we see.
//
// (This file is auto-generated. Do not edit it directly! To apply
// changes you need to modify `./internal/tutorial/measure/chapter01/main.go`.)
//
// ## main.go
//
// We declare the package and import useful libraries. The most
// important library we're importing here is, of course, `internal/measure`.
//
// ```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measure"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	// ```
	//
	// Now we define command line flags useful to test this program. We use
	// the `flags` package for that. We want the user to be able to configure
	// both the domain name to resolve and the resolution timeout.
	//
	// We call `flag.Parse` to parse the CLI flags.
	//
	// We also configure the logger to emit debug messages.
	//
	// ```Go
	domain := flag.String("domain", "example.com", "domain to resolve")
	timeout := flag.Duration("timeout", 4*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	// ```
	//
	// We create a context and we attach a timeout to it. This is a pretty
	// standard way to configure a timeout in Go.
	//
	// ```Go
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	// ```
	//
	// We record the time when we started measuring. Most code inside
	// `internal/measure` requires this information to emit the relative
	// time after which specific events occurred.
	//
	// We also create a `Trace`, i.e., a data structure that collects
	// I/O events generated by TCP and UDP connections.
	//
	// ```Go
	begin := time.Now()
	trace := measure.NewTrace(begin)
	// ```
	//
	// Now we create a `Measurer`. The procedure for creating a mx
	// is the same regardless of what you want to use it for.
	//
	// All the fields of a mx must be initialized. These are the
	// fields we are setting and why we need them:
	//
	// - Begin is the time when we started measuring. We use this field
	// to generate measurement records containing timing information.
	//
	// - Logger emits logs. You cannot opt-out of providing a logger
	// because having a logger is the common case.
	//
	// - Connector is the data structure that implements connecting and
	// needs to now about the beginning of time, the logger, and the
	// trace. The connector uses the trace to wrap newly created TCP/UDP
	// net.Conn instances and setup I/O events tracing.
	//
	// - TLSHandshaker is the data structure that, given a TCP conn and
	// a suitable config, knows how to perform a TLS handshake.
	//
	// - QUICHandshaker is the data structure that, given an endpoint and
	// a suitable config, knows how to perform a QUIC handshake. It uses
	// the Trace to wrap UDP connections and setup I/O events tracing.
	//
	// - Trace is the trace, which we need to pass to private structs
	// created when we run specific flows.
	//
	// Connector, TLSHandshaker, and QUICHandshaker are interfaces, which
	// allows for mocking, testing, and generally wrapping them.
	//
	// Begin, Logger, and Trace are either used directly or passed to
	// private data structures created when we run specific flows.
	//
	// The Measurer has not specifically been designed with concurrency
	// in mind. In particular, you are supposed to not modify any of
	// its fields after initialization.
	//
	// ```Go
	mx := &measure.Measurer{
		Begin:          begin,
		Logger:         log.Log,
		Connector:      measure.NewConnector(begin, log.Log, trace),
		TLSHandshaker:  measure.NewTLSHandshakerStdlib(begin, log.Log),
		QUICHandshaker: measure.NewQUICHandshaker(begin, log.Log, trace),
		Trace:          trace,
	}
	// ```
	//
	// We now use the measurer to perform the DNS lookup of the given
	// domain using the system resolver.
	//
	// The `ctx` argument guarantees that there is a timeout. The `*domain`
	// argument contains the domain we want to resolve.
	//
	// ```Go
	m := mx.LookupHostSystem(ctx, *domain)
	// ```
	//
	// The result, `m`, contains the result of measuring the DNS lookup
	// operation. We are not going to go into the details of this
	// data structure here. Any description that is not part of the
	// code may become stale.
	//
	// Though, this is perhaps a good place to mention how to get
	// documentation from the command line using `go doc`.
	//
	// You can read all the `./internal/measure` documentation using:
	//
	// ```bash
	// go doc -all ./internal/measure
	// ```
	//
	// You can get specific documentation for `Measurer` with
	//
	// ```bash
	// go doc -all ./internal/measure.Measurer
	// ```
	//
	// And you can narrow it down by reading about `LookupHostSystem` with:
	//
	// ```bash
	// go doc -all ./internal/measure.Measurer.LookupHostSystem
	// ```
	//
	// The rest of `main` serializes `m` to JSON and prints it. We are
	// going to run this test program with a couple of test cases to see
	// how its output changes.
	//
	// This is probably a good place to point out that the data
	// format that you will see here is not necessarily be the
	// data format submitted to the OONI backend. This data
	// format is the one specific to `internal/measure`. We will
	// add a chapter regarding converting to the OONI data
	// format to this tutorial once we have written code for that.
	//
	// As a final note, the `PanicOnError` is here because the
	// message `m` *can* be marshalled to JSON. It still feels a
	// bit better having an assertion for our assumptions than
	// outrightly ignoring the error code. (We tend to use such
	// a convention quite frequently in the OONI codebase.)
	//
	// ```Go
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

// ```
//
// ## Running the example program
//
// Let us run the program with default arguments first. You can do
// this operation by running:
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter01
// ```
//
// If you do that you obtain some logging messages, which are out of
// the scope of this tutorial, and the following JSON:
//
// ```JSON
// {
//   "engine": "system",
//   "domain": "example.com",
//   "started": 3125,
//   "completed": 1597875,
//   "failure": null,
//   "addrs": [
//     "93.184.216.34",
//     "2606:2800:220:1:248:1893:25c8:1946"
//   ]
// }
// ```
//
// This message tells us:
//
// - that we are using the "system" resolver;
//
// - that we wanted to resolve the "example.com" domain;
//
// - that the DNS resolution started 3,125 nanoseconds
//   into the life of the program;
//
// - that the DNS resolution completed 1,597,875
//   nanoseconds into the life of the program (hence
//   the resolution took ~1.4 ms);
//
// - that the *failure* is *null* (i.e., no error
//   has actually occurred);
//
// - that the resolution discovered two addresses: an
//   IPv4 address and an IPv6 address.
//
// Let us now change the domain to resolve to be `antani.ooni.org`,
// which we can do by running this command:
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter01 -domain antani.ooni.org
// ```
//
// This is the output JSON:
//
// ```JSON
// {
//   "engine": "system",
//   "domain": "antani.ooni.org",
//   "started": 2708,
//   "completed": 68244417,
//   "failure": "dns_nxdomain_error",
//   "addrs": null
// }
// ```
//
// What changes here? Well, we see that now the resolution took ~68 ms
// and that there is a failure indicating NXDOMAIN. Also, we have obviously
// not resolved any IP addresses hence `addrs` is `null`.
//
// The failure indicated there is just one of the failures that a OONI
// Probe may return (see [df-007-errors](https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md)).
//
// Let us now try with an insanely low timeout:
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter01 -timeout 250us
// ```
//
// To get this JSON:
//
// ```JSON
// {
//  "engine": "system",
//  "domain": "example.com",
//  "started": 1833,
//  "completed": 533500,
//  "failure": "generic_timeout_error",
//  "addrs": null
// }
// ```
//
// We see that here we completed in ~500 microseconds, which is twice the
// timeout we specified (timeouts are not perfect, especially when you
// configure unreasonably low timeouts).
//
// ## Conclusions
//
// This is it. We have seen how to measure the *system resolver* and we have
// also seen which easy-to-provoke errors we can get.
//
// -=-=- StopHere -=-=-
