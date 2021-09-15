
# Chapter II: establishing TCP connections

In this chapter we explain how to measure establishing TCP connections.

We will first write a simple `main.go` file that shows how to use
this functionality. Then, we will show some runs of this file, and
we will comment the output that we see.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measure/chapter02/main.go`.)

## main.go

We declare the package and import useful libraries. The most
important library we're importing here is, of course, `internal/measure`.

```Go
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
```
### Setup

This first part of `main.go` is really similar to the previous
chapter, so there is not much new to say here.

```Go
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 4*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	begin := time.Now()
	trace := measure.NewTrace(begin)
```

We create a `Measurer` like we did in the previous chapter.

```Go
	mx := &measure.Measurer{
		Begin:          begin,
		Logger:         log.Log,
		Connector:      measure.NewConnector(begin, log.Log, trace),
		TLSHandshaker:  measure.NewTLSHandshakerStdlib(begin, log.Log),
		QUICHandshaker: measure.NewQUICHandshaker(begin, log.Log, trace),
		Trace:          trace,
	}
```

### Measurer.TCPConnect flow

We then call `TCPConnect`, which executes the connect *flow*. The
input is the context (for timeouts), and the address of the
endpoint to which we want to connect.

```Go
	m := mx.TCPConnect(ctx, *address)
```

The result is a message `m` that we are going to print later.

### Printing the results

We don't need to worry about closing connections. The `TCPConnect`
flow closes the connection for us. (If you need the connection
to be open at this point, then you should perhaps use another flow
or write a flow that fullfills your needs.)

The rest of the code just prints `m`.

```Go
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## Running the example program

Let us run the program with default arguments first. You can do
this operation by running:

```bash
go run -race ./internal/tutorial/measure/chapter02
```

Here is the JSON we obtain in output:

```JSON
{
  "network": "tcp",
  "address": "8.8.4.4:443",
  "started": 2750,
  "completed": 24350166,
  "failure": null
}
```

This is what it says:

- we are connecting a "tcp" socket;

- the destination address is "8.8.4.4:443";

- connect started 2750 nanoseconds into the program life;

- connect terminated 24,350,166 nanoseconds (~24 ms)
  into the program life;

- the operation succeeded (`failure` is `nil`).

Let us now see if we can provoke some errors and timeouts.

### Measurement with connection refused

Let us start with an IP address where there's no listening socket:

```bash
go run -race ./internal/tutorial/measure/chapter02 -address 127.0.0.1:1
```

We get this JSON:

```JSON
{
  "network": "tcp",
  "address": "127.0.0.1:1",
  "started": 2667,
  "completed": 675500,
  "failure": "connection_refused"
}
```

And here's an error telling us the connection was refused.

### Measurement with timeouts

Let us now try to obtain a timeout:

```bash
go run -race ./internal/tutorial/measure/chapter02 -address 8.8.4.4:1
```

We get this JSON:

```JSON
{
  "network": "tcp",
  "address": "8.8.4.4:1",
  "started": 3208,
  "completed": 4005731000,
  "failure": "generic_timeout_error"
}
```

So, we clearly see that our 4 seconds timeout is working.

Let us now use a very large timeout:

```bash
go run -race ./internal/tutorial/measure/chapter02 -address 8.8.4.4:1 -timeout 1h
```

To get this JSON:

```JSON
{
  "network": "tcp",
  "address": "8.8.4.4:1",
  "started": 3000,
  "completed": 15007059167,
  "failure": "generic_timeout_error"
}
```

We see a timeout after 15s. This is the connector's one watchdog
timeout (we don't want to have unbounded connect times).

## Conclusions

We have seen how to measure the operation of connecting
to a specific TCP endpoint.

