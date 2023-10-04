
# Chapter III: using a custom DNS-over-UDP resolver

In this chapter we learn how to measure sending DNS queries to
a DNS server speaking the DNS-over-UDP protocol.

Without further ado, let's describe our example `main.go` program
and let's use it to better understand this flow.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter03/main.go`.)

## main.go

The initial part of the program is pretty much the same as the one
used in previous chapters, so I will not add further comments.

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
	query := flag.String("query", "example.com", "domain to resolve")
	address := flag.String("address", "8.8.4.4:53", "DNS-over-UDP server address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
```

### Using a custom UDP resolver

We now invoke `LookupHostUDP`. We specify:

- a context for timeout information;

- the domain to query for;

- the address of the DNS-over-UDP server endpoint.

```Go
	m := mx.LookupHostUDP(ctx, *query, *address)
```

Also this operation returns a measurement, which
we print using the usual three-liner.

```Go
	data, err := json.Marshal(measurex.NewArchivalDNSMeasurement(m))
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## Running the example program

As before, let us start off with a vanilla run:

```bash
go run -race ./internal/tutorial/measurex/chapter03 | jq
```

This time we get a much larger JSON, so I will pretend it is
actually JavaScript and add comments to explain it inline.

(This is the first case in which we see how a single
method call for measurer causes several events to
be generated and inserted into a `Measurement`.)

```JavaScript
{
  "domain": "example.com",

  // This block shows the read and write events
  // occurred on the sockets (because we control
  // in full the implementation of this DNS
  // over UDP resolver, we can see these events)
  //
  // See https://github.com/ooni/spec/blob/master/data-formats/df-008-netevents.md
  // for a description of this data format.
  "network_events": [
    {
      "address": "8.8.4.4:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "write",
      "proto": "udp",
      "t": 0.00048825,
      "started": 0.000462917,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:53",
      "failure": null,
      "num_bytes": 45,
      "operation": "read",
      "proto": "udp",
      "t": 0.022081833,
      "started": 0.000502625,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "write",
      "proto": "udp",
      "t": 0.022433083,
      "started": 0.022423875,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:53",
      "failure": null,
      "num_bytes": 57,
      "operation": "read",
      "proto": "udp",
      "t": 0.046706,
      "started": 0.022443833,
      "oddity": ""
    }
  ],

  // This block shows the query we sent (encoded as base64)
  // and the response we received. Here we clearly see
  // that we perform two DNS "round trip" (i.e., send request
  // and receive response) to resolve a domain: one for
  // A and the other for AAAA.
  //
  // We don't have a specification for this data format yet.
  "dns_events": [
    {
      "engine": "udp",
      "resolver_address": "8.8.4.4:53",
      "raw_query": {
        "data": "dGwBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
        "format": "base64"
      },
      "started": 0.000205083,
      "t": 0.022141333,
      "failure": null,
      "raw_reply": {
        "data": "dGyBgAABAAEAAAAAB2V4YW1wbGUDY29tAAABAAHADAABAAEAAEuIAARduNgi",
        "format": "base64"
      }
    },
    {
      "engine": "udp",
      "resolver_address": "8.8.4.4:53",
      "raw_query": {
        "data": "Ts8BAAABAAAAAAAAB2V4YW1wbGUDY29tAAAcAAE=",
        "format": "base64"
      },
      "started": 0.022221417,
      "t": 0.046733125,
      "failure": null,
      "raw_reply": {
        "data": "Ts+BgAABAAEAAAAAB2V4YW1wbGUDY29tAAAcAAHADAAcAAEAAFOQABAmBigAAiAAAQJIGJMlyBlG",
        "format": "base64"
      }
    }
  ],

  // This is the same kind of result as before, we
  // show the emitted queries and the resolved addrs.
  //
  // Also note how here the resolver_address is the
  // correct endpoint address and the engine tells us
  // that we're using DNS over UDP.
  "queries": [
    {
      "answers": [
        {
          "answer_type": "A",
          "ipv4": "93.184.216.34"
        }
      ],
      "engine": "udp",
      "failure": null,
      "hostname": "example.com",
      "query_type": "A",
      "resolver_address": "8.8.4.4:53",
      "t": 0.046766833,
      "started": 0.000124375,
      "oddity": ""
    },
    {
      "answers": [
        {
          "answer_type": "AAAA",
          "ivp6": "2606:2800:220:1:248:1893:25c8:1946"
        }
      ],
      "engine": "udp",
      "failure": null,
      "hostname": "example.com",
      "query_type": "AAAA",
      "resolver_address": "8.8.4.4:53",
      "t": 0.046766833,
      "started": 0.000124375,
      "oddity": ""
    }
  ]
}
```

This data format is really an extension of the `LookupHostSystem`
one. It just adds more fields that clarify what happened at low
level in terms of socket I/O and queries sent and received.

Let us now try to provoke some errors and see how the
output JSON changes because of them.

### Measurement with NXDOMAIN

Let us try to get a NXDOMAIN error.

```bash
go run -race ./internal/tutorial/measurex/chapter03 -query antani.ooni.org | jq
```

This produces the following JSON:

```JavaScript
{
  "domain": "antani.ooni.org",
  "network_events": [ /* snip */ ],

  "dns_events": [
    {
      "engine": "udp",
      "resolver_address": "8.8.4.4:53",
      "raw_query": {
        "data": "p7YBAAABAAAAAAAABmFudGFuaQRvb25pA29yZwAAAQAB",
        "format": "base64"
      },
      "started": 0.000152959,
      "t": 0.051650917,
      "failure": null,
      "raw_reply": {
        "data": "p7aBgwABAAAAAQAABmFudGFuaQRvb25pA29yZwAAAQABwBMABgABAAAHCAA9BGRuczERcmVnaXN0cmFyLXNlcnZlcnMDY29tAApob3N0bWFzdGVywDJhbwqOAACowAAADhAACTqAAAAOEQ==",
        "format": "base64"
      }
    },
    {
      "engine": "udp",
      "resolver_address": "8.8.4.4:53",
      "raw_query": {
        "data": "ILkBAAABAAAAAAAABmFudGFuaQRvb25pA29yZwAAHAAB",
        "format": "base64"
      },
      "started": 0.051755209,
      "t": 0.101094375,
      "failure": null,
      "raw_reply": {
        "data": "ILmBgwABAAAAAQAABmFudGFuaQRvb25pA29yZwAAHAABwBMABgABAAAHCAA9BGRuczERcmVnaXN0cmFyLXNlcnZlcnMDY29tAApob3N0bWFzdGVywDJhbwqOAACowAAADhAACTqAAAAOEQ==",
        "format": "base64"
      }
    }
  ],

  "queries": [
    {
      "answers": null,
      "engine": "udp",
      "failure": "dns_nxdomain_error",
      "hostname": "antani.ooni.org",
      "query_type": "A",
      "resolver_address": "8.8.4.4:53",
      "t": 0.101241667,
      "started": 8.8e-05,
      "oddity": "dns.lookup.nxdomain"
    },
    {
      "answers": null,
      "engine": "udp",
      "failure": "dns_nxdomain_error",
      "hostname": "antani.ooni.org",
      "query_type": "AAAA",
      "resolver_address": "8.8.4.4:53",
      "t": 0.101241667,
      "started": 8.8e-05,
      "oddity": "dns.lookup.nxdomain"
    }
  ]
}
```

We indeed get a NXDOMAIN error as the failure in `lookup_host`.

Let us now decode one of the replies by using this program:

```
package main

import (
    "fmt"
    "encoding/base64"

    "github.com/miekg/dns"
)

func main() {
    const query = "azGBgwABAAAAAQAABmFudGFuaQRvb25pA29yZwAAHAABwBMABgABAAAHCAA9BGRuczERcmVnaXN0cmFyLXNlcnZlcnMDY29tAApob3N0bWFzdGVywDJhABz8AACowAAADhAACTqAAAAOEQ=="
    data, _ := base64.StdEncoding.DecodeString(query)
    msg := new(dns.Msg)
    _ = msg.Unpack(data)
    fmt.Printf("%s\n", msg)
}
```

where `query` is one of the replies. If we run this program
we get as the output:

```
;; opcode: QUERY, status: NXDOMAIN, id: 27441
;; flags: qr rd ra; QUERY: 1, ANSWER: 0, AUTHORITY: 1, ADDITIONAL: 0

;; QUESTION SECTION:
;antani.ooni.org.	IN	 AAAA

;; AUTHORITY SECTION:
ooni.org.	1800	IN	SOA	dns1.registrar-servers.com. hostmaster.registrar-servers.com. 1627397372 43200 3600 604800 3601
```

### Measurement with timeout

Let us now query an IP address known for not responding
to DNS queries, to get a timeout.

```bash
go run -race ./internal/tutorial/measurex/chapter03 -address 182.92.22.222:53
```

Here's the corresponding JSON:

```JavaScript
{
  "domain": "example.com",
  "network_events": [
    {
      "address": "182.92.22.222:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "write",
      "proto": "udp",
      "t": 0.000479583,
      "started": 0.00045525,
      "oddity": ""
    },
    {
      "address": "182.92.22.222:53",
      "failure": "generic_timeout_error",  /* <--- */
      "operation": "read",
      "proto": "udp",
      "t": 5.006016292,
      "started": 0.000491792,
      "oddity": ""
    }
  ],
  "dns_events": [
    {
      "engine": "udp",
      "resolver_address": "182.92.22.222:53",
      "raw_query": {
        "data": "GRUBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
        "format": "base64"
      },
      "started": 0.00018225,
      "t": 5.006185667,
      "failure": "generic_timeout_error",   /* <--- */
      "raw_reply": null
    }
    /* snip */
  ],
  "queries": [
    {
      "answers": null,
      "engine": "udp",
      "failure": "generic_timeout_error",   /* <--- */
      "hostname": "example.com",
      "query_type": "A",
      "resolver_address": "182.92.22.222:53",
      "t": 5.007385458,
      "started": 0.000107583,
      "oddity": "dns.lookup.timeout"
    },
    {
      "answers": null,
      "engine": "udp",
      "failure": "generic_timeout_error",   /* <--- */
      "hostname": "example.com",
      "query_type": "AAAA",
      "resolver_address": "182.92.22.222:53",
      "t": 5.007385458,
      "started": 0.000107583,
      "oddity": "dns.lookup.timeout"
    }
  ]
}
```

We see that we fail with a timeout (I have marked some of them
with comments inside the JSON). We see the timeout at three different
levels of abstractions (from lower to higher abstraction): at the socket
layer (`network_events`), during the DNS round trip (`dns_events`),
during the DNS lookup (`queries`).

What we also see is that `t`'s value is ~5s when the `read` event
fails, which tells us about the socket's read timeout.

### Measurement with REFUSED error

Let us now try to get a REFUSED DNS Rcode, again from servers
that are, let's say, kind enough to easily help.

```bash
go run -race ./internal/tutorial/measurex/chapter03 -address 180.97.36.63:53 | jq
```

Here's the answer I get:

```JavaScript
{
  "domain": "example.com",

  // The network events look normal this time
  "network_events": [
    {
      "address": "180.97.36.63:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "write",
      "proto": "udp",
      "t": 0.000492125,
      "started": 0.000467042,
      "oddity": ""
    },
    {
      "address": "180.97.36.63:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "read",
      "proto": "udp",
      "t": 0.321373542,
      "started": 0.000504833,
      "oddity": ""
    },
    {
      "address": "180.97.36.63:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "write",
      "proto": "udp",
      "t": 0.322500875,
      "started": 0.322450042,
      "oddity": ""
    },
    {
      "address": "180.97.36.63:53",
      "failure": null,
      "num_bytes": 29,
      "operation": "read",
      "proto": "udp",
      "t": 0.655514542,
      "started": 0.322557667,
      "oddity": ""
    }
  ],

  // Exercise: do like I did before and decode the messages
  "dns_events": [
    {
      "engine": "udp",
      "resolver_address": "180.97.36.63:53",
      "raw_query": {
        "data": "WcgBAAABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
        "format": "base64"
      },
      "started": 0.000209875,
      "t": 0.321504042,
      "failure": null,
      "raw_reply": {
        "data": "WciBBQABAAAAAAAAB2V4YW1wbGUDY29tAAABAAE=",
        "format": "base64"
      }
    },
    {
      "engine": "udp",
      "resolver_address": "180.97.36.63:53",
      "raw_query": {
        "data": "I9oBAAABAAAAAAAAB2V4YW1wbGUDY29tAAAcAAE=",
        "format": "base64"
      },
      "started": 0.321672042,
      "t": 0.655680792,
      "failure": null,
      "raw_reply": {
        "data": "I9qBBQABAAAAAAAAB2V4YW1wbGUDY29tAAAcAAE=",
        "format": "base64"
      }
    }
  ],

  // We see both in the error and in the oddity
  // that the response was "REFUSED"
  "queries": [
    {
      "answers": null,
      "engine": "udp",
      "failure": "dns_refused_error",
      "hostname": "example.com",
      "query_type": "A",
      "resolver_address": "180.97.36.63:53",
      "t": 0.655814875,
      "started": 0.000107417,
      "oddity": "dns.lookup.refused"
    },
    {
      "answers": null,
      "engine": "udp",
      "failure": "dns_refused_error",
      "hostname": "example.com",
      "query_type": "AAAA",
      "resolver_address": "180.97.36.63:53",
      "t": 0.655814875,
      "started": 0.000107417,
      "oddity": "dns.lookup.refused"
    }
  ]
}
```

## Conclusion

We have seen how to send DNS queries over UDP, measure the
results, and what happens on common error conditions.

