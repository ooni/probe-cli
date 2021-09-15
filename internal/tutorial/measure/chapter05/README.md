
# Chapter V: QUIC handshaking

This chapter describes the *QUIC handshake flow*. This flow produces
measurements of the QUIC handshake.

Without further ado, let's describe our example `main.go` program
and let's use it to better understand this flow.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measure/chapter05/main.go`.)

## main.go

The initial part of the program is pretty much the same as the one
used in previous chapters, so I will not add further comments.

```Go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measure"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	sni := flag.String("sni", "dns.google", "SNI to use")
	timeout := flag.Duration("timeout", 4*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	begin := time.Now()
	trace := measure.NewTrace(begin)
	mx := measure.NewMeasurerStdlib(begin, log.Log, trace)
```

The main difference compared to the previous chapter is that
QUIC combines connecting and handshaking into the same operation,
so the arguments are: the context for timeouts, the address of
the UDP endpoint, the TLS configuration.

There are no other significant differences in the program.

```Go
	m := mx.QUICEndpointDial(ctx, *address, &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h3"},
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## Running the example program

As before, let us start off with a vanilla run:

```bash
go run ./internal/tutorial/measure/chapter05
```

Produces this JSON:

```JavaScript
{
  // This is the data structure produced by the QUIC
  // handshaker abstraction we created above. It's very
  // similar to the TLS handshake data structure.
  "quic_handshake": {
    "address": "8.8.4.4:443",
    "config": {
      "sni": "dns.google",
      "alpn": [
        "h3"
      ],
      "no_tls_verify": false
    },
    "started": 12031,
    "completed": 46255736,
    "failure": null,
    "connection_state": {
      "tls_version": "TLSv1.3",
      "cipher_suite": "TLS_AES_128_GCM_SHA256",
      "negotiated_protocol": "h3",
      "peer_certificates": [
        "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"
      ]
    }
  },

  // This is the list of network events. Because QUIC does
  // not connect UDP sockets, we have read_from and write_to
  // operations instead of read and write operations.
  "network_events": [
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 1312518,
      "completed": 1368142,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 547619,
      "completed": 27662685,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 28157926,
      "completed": 28211560,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 27681722,
      "completed": 43609601,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 43627794,
      "completed": 43648540,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 43654491,
      "completed": 43816700,
      "failure": null,
      "num_bytes": 1252
    },
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 43923164,
      "completed": 43977783,
      "failure": null,
      "num_bytes": 44
    },
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 44020562,
      "completed": 44037297,
      "failure": null,
      "num_bytes": 44
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 43835285,
      "completed": 44211377,
      "failure": null,
      "num_bytes": 1235
    },
    {
      "operation": "read_from",
      "address": "8.8.4.4:443",
      "started": 44219860,
      "completed": 44223519,
      "failure": null,
      "num_bytes": 68
    },
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 46127440,
      "completed": 46148869,
      "failure": null,
      "num_bytes": 83
    },
    {
      "operation": "write_to",
      "address": "8.8.4.4:443",
      "started": 46177174,
      "completed": 46211319,
      "failure": null,
      "num_bytes": 33
    }
  ]
}
```

A future version of this document will describe what
happens under several QUIC error conditions.

## Conclusion

We have seen how to use the QUIC handshake flow to
measure what happens during a QUIC handshake.

