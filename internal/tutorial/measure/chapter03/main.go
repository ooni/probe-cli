// -=-=- StartHere -=-=-
//
// # Chapter III: using a custom DNS-over-UDP resolver
//
// In this chapter we learn how to measure sending DNS queries to
// a DNS server speaking the DNS-over-UDP protocol.
//
// Without further ado, let's describe our example `main.go` program
// and let's use it to better understand this flow.
//
// (This file is auto-generated. Do not edit it directly! To apply
// changes you need to modify `./internal/tutorial/measure/chapter03/main.go`.)
//
// ## main.go
//
// The initial part of the program is pretty much the same as the one
// used in previous chapters, so I will not add further comments.
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
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/measure"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	address := flag.String("address", "8.8.4.4:53", "remote endpoint address")
	query := flag.String("query", "example.com", "DNS query to send")
	timeout := flag.Duration("timeout", 4*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	begin := time.Now()
	// ```
	//
	// ### Creating a Measurer using a factory
	//
	// This time, rather than manually filling all the fields of the
	// `Measurer` like we did before, we call a factory function that
	// is equivalent to the many-liner we wrote previously.
	//
	// ```Go
	mx := measure.NewMeasurerStdlib(begin, log.Log)
	// ```
	//
	// The advantage of this factory is that the code is less
	// verbose. You can still modify fields _before usage_ if you
	// need to change any of them.
	//
	// Also, this factory creates a trace for the measurer
	// automatically. The suggested usage is to use a
	// measurer for a single measurement. So you do not
	// have to worry about overlapping events in the trace.
	//
	// ### The Measurer.LookupHostUDP flow
	//
	// We now invoke the `LookupHostUDP` flow. We specify:
	//
	// - a context for timeout information;
	//
	// - the domain to query for;
	//
	// - the type of the query (in this case A);
	//
	// - the address of the DNS-over-UDP server endpoint.
	//
	// ```Go
	m := mx.LookupHostUDP(ctx, *query, dns.TypeA, *address)
	// ```
	//
	// This flow returns the same data type as `LookupHostSystem` though
	// `LookupHostSystem` queries for `A` and `AAAA` together. In the
	// future there may be a compound flow that queries both `A` and `AAAA`
	// toghether using an UDP resolver. For now this seems unnecessary.
	//
	// Likewise, the system resolver may implement caching and retries
	// while `LookupHostUDP` does not. It just sends a query once and
	// returns the response (if any) or a timeout error.
	//
	// ### Printing the results
	//
	// Let us now print this message like we did before.
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
// As before, let us start off with a vanilla run:
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter03
// ```
//
// This time we get a much larger JSON, so I will pretend it is
// actually JavaScript and add comments to explain it inline.
//
// ```JavaScript
// {
//   // This tells us we're using a DNS-over-UDP resolver
//   "engine": "udp",
//
//   // This is the address of the server UDP endpoint
//   "address": "8.8.4.4:53",
//
//   // This indicates the query type
//   "query_type": "A",
//
//   // These fields are also present in the result of LookupHostSystem
//   "domain": "example.com",
//   "started": 605446,
//   "completed": 20723542,
//
//   // This is the raw query message.
//   "query": "XiIBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
//
//   // This indicates whether there was an error.
//   "failure": null,
//
//   // This contains the result addresses
//   "addrs": [
//     "93.184.216.34"
//   ],
//
//   // This is the raw reply message.
//   "reply": "XiKBgAABAAEAAAAAB2V4YW1wbGUDY29tAAABAAHADAABAAEAAFRDAARduNgi",
//
//   // This is the trace created by `Trace`. We record every
//   // network I/O operation occurring on the UDP socket.
//   "network_events": [
//     {
//       "operation": "write",
//       "address": "8.8.4.4:53",
//       "started": 935284,
//       "completed": 1012331,
//       "failure": null,
//       "num_bytes": 29
//     },
//     {
//       "operation": "read",
//       "address": "8.8.4.4:53",
//       "started": 1048750,
//       "completed": 20665746,
//       "failure": null,
//       "num_bytes": 45
//     }
//   ]
// }
// ```
//
// This data format is really an extension of the `LookupHostSystem`
// one. It just adds more fields that clarify what happened at low
// level. Also, `LookupHostSystem` will put both IPv4 and IPv6
// addresses into a single message, while `LookupHostUDP` will only
// put IPv4 when querying for `A` and IPv6 for `AAAA`.
//
// Let us now try to provoke some errors and see how the
// output JSON changes because of them.
//
// ### Measurement with NXDOMAIN
//
// Let us first try to get a NXDOMAIN error.
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter03 -query antani.ooni.org
// ```
//
// This produces the following JSON:
//
// ```JSON
// {
//   "engine": "udp",
//   "address": "8.8.4.4:53",
//   "query_type": "A",
//   "domain": "antani.ooni.org",
//   "started": 579109,
//   "completed": 39232278,
//   "query": "RQEBAAABAAAAAAAABmFudGFuaQRvb25pA29yZwAAAQAB",
//   "failure": "dns_nxdomain_error",
//   "addrs": null,
//   "reply": "RQGBgwABAAAAAQAABmFudGFuaQRvb25pA29yZwAAAQABwBMABgABAAAHCAA9BGRuczERcmVnaXN0cmFyLXNlcnZlcnMDY29tAApob3N0bWFzdGVywDJhABz8AACowAAADhAACTqAAAAOEQ==",
//   "network_events": [
//     {
//       "operation": "write",
//       "address": "8.8.4.4:53",
//       "started": 910099,
//       "completed": 1053411,
//       "failure": null,
//       "num_bytes": 33
//     },
//     {
//       "operation": "read",
//       "address": "8.8.4.4:53",
//       "started": 1091463,
//       "completed": 39175093,
//       "failure": null,
//       "num_bytes": 106
//     }
//   ]
// }
// ```
//
// We indeed get a NXDOMAIN error as the failure.
//
// ### Measurement with timeout
//
// Let us now query an IP address known for not responding
// to DNS queries, to get a timeout.
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter03 -address 182.92.22.222:53
// ```
//
// Here's the corresponding JSON:
//
// ```JSON
// {
//   "engine": "udp",
//   "address": "182.92.22.222:53",
//   "query_type": "A",
//   "domain": "example.com",
//   "started": 560639,
//   "completed": 4003970851,
//   "query": "mUUBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
//   "failure": "generic_timeout_error",
//   "addrs": null,
//   "network_events": [
//     {
//       "operation": "write",
//       "address": "182.92.22.222:53",
//       "started": 915671,
//       "completed": 985163,
//       "failure": null,
//       "num_bytes": 29
//     }
//   ]
// }
// ```
//
// We see that we do fail with a timeout. We also see that there is a
// "write" event but there is no corresponding "read" (this may change
// in a future release of `internal/measure`). Why? Well, the fact is
// that the transport immediately terminates once the context says there
// is a timeout, leaving the read pending in the background until we
// have the watchdog timeout for waiting for DNS replies.
//
// To see the DNS-replies watchdog timeout in action, we need to
// inflate significantly the timeout set from command line, so that
// the watchdog timeout for DNS UDP sockets could kick in.
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter03 -address 182.92.22.222:53 -timeout 1h
// ```
//
// And here's what we see:
//
// ```JSON
// {
//   "engine": "udp",
//   "address": "182.92.22.222:53",
//   "query_type": "A",
//   "domain": "example.com",
//   "started": 541738,
//   "completed": 5006073519,
//   "query": "g7EBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
//   "failure": "generic_timeout_error",
//   "addrs": null,
//   "network_events": [
//     {
//       "operation": "write",
//       "address": "182.92.22.222:53",
//       "started": 857685,
//       "completed": 940615,
//       "failure": null,
//       "num_bytes": 29
//     },
//     {
//       "operation": "read",
//       "address": "182.92.22.222:53",
//       "started": 982556,
//       "completed": 5006000271,
//       "failure": "generic_timeout_error",
//       "num_bytes": 0
//     }
//   ]
// }
// ```
//
// So we see that the watchdog timeout for the UDP socket
// defaults to five seconds.
//
// ### Measurement with REFUSED error
//
// Let us now try to get a REFUSED DNS Rcode, again from servers
// that are, let's say, kind enough to easily help.
//
// ```bash
// go run -race ./internal/tutorial/measure/chapter03 -address 180.97.36.63:53
// ```
//
// Here's the answer I get:
//
// ```JSON
// {
//   "engine": "udp",
//   "address": "180.97.36.63:53",
//   "query_type": "A",
//   "domain": "example.com",
//   "started": 600830,
//   "completed": 540762023,
//   "query": "0YsBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
//   "failure": "dns_refused_error",
//   "addrs": null,
//   "reply": "0YuBBQABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
//   "network_events": [
//     {
//       "operation": "write",
//       "address": "180.97.36.63:53",
//       "started": 875084,
//       "completed": 1023665,
//       "failure": null,
//       "num_bytes": 29
//     },
//     {
//       "operation": "read",
//       "address": "180.97.36.63:53",
//       "started": 1068502,
//       "completed": 540693264,
//       "failure": null,
//       "num_bytes": 29
//     }
//   ]
// }
// ```
//
// ## Conclusion
//
// We have seen how we can configure and use the flow for
// sending DNS queries over UDP and we have seen some common errors.
//
// -=-=- StopHere -=-=-
