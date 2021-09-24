// -=-=- StartHere -=-=-
//
// # Chapter II: establishing TCP connections
//
// In this chapter we explain how to measure establishing TCP connections.
//
// We will first write a simple `main.go` file that shows how to use
// this functionality. Then, we will show some runs of this file, and
// we will comment the output that we see.
//
// (This file is auto-generated. Do not edit it directly! To apply
// changes you need to modify `./internal/tutorial/measurex/chapter02/main.go`.)
//
// ## main.go
//
// We declare the package and import useful packages. The most
// important package we're importing here is, of course, `internal/measurex`.
//
// ```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	// ```
	// ### Setup
	//
	// This first part of `main.go` is really similar to the previous
	// chapter, so there is not much new to say here.
	//
	// ```Go
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	// ```
	//
	// ### Creaging a Measurer
	//
	// We create a `Measurer` like we did in the previous chapter.
	//
	// ```Go
	mx := measurex.NewMeasurerWithDefaultSettings()
	// ```
	//
	// ### Establishing a TCP connection.
	//
	// We then call `TCPConnect`, which establishes a connection
	// and returns the corresponding measurent.
	//
	// The arguments are the context (for timeouts), and the address
	// of the endpoint to which we want to connect. (Here and in
	// most of this tutorial with "endpoint" we mean an IP address
	// and a port, serialized as "ADDRESS:PORT", where the
	// address is quoted with "[" and "]" if IPv6, e.g., `[::1]:53`.)
	//
	// ```Go
	m := mx.TCPConnect(ctx, *address)
	// ```
	//
	// ### Printing the measurement
	//
	// The rest of the main function is just like in the previous chapter.
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
// go run -race ./internal/tutorial/measurex/chapter02
// ```
//
// Here is the JSON we obtain in output:
//
// ```JavaScript
// {
//   // These two fields identify the endpoint
//   "network": "tcp",
//   "address": "8.8.4.4:443",
//
//   // This block contains the results of the connect syscall
//   // using the df-008-netevents data format.
//   "connect": [
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "operation": "connect",
//       "proto": "tcp",
//       "t": 0.026879041,
//       "started": 8.8625e-05,
//       "oddity": ""
//     }
//   ]
// }
// ```
//
// This is what it says:
//
// - we are connecting a "tcp" socket;
//
// - the destination endpoint address is "8.8.4.4:443";
//
// - connect terminated ~0.027 seconds into the program life;
//
// - the operation succeeded (`failure` is `nil`).
//
// Let us now see if we can provoke some errors and timeouts.
//
// ### Measurement with connection refused
//
// Let us start with an IP address where there's no listening socket:
//
// ```bash
// go run -race ./internal/tutorial/measurex/chapter02 -address 127.0.0.1:1
// ```
//
// We get this JSON:
//
// ```JSON
// {
//   "network": "tcp",
//   "address": "127.0.0.1:1",
//   "connect": [
//     {
//       "address": "127.0.0.1:1",
//       "failure": "connection_refused",
//       "operation": "connect",
//       "proto": "tcp",
//       "t": 0.000372167,
//       "started": 8.4917e-05,
//       "oddity": "tcp.connect.refused"
//     }
//   ]
// }
//
// ```
//
// And here's an error telling us the connection was refused and
// the oddity that classifies the error.
//
// ### Measurement with timeouts
//
// Let us now try to obtain a timeout:
//
// ```bash
// go run -race ./internal/tutorial/measurex/chapter02 -address 8.8.4.4:1
// ```
//
// We get this JSON:
//
// ```JSON
// {
//   "network": "tcp",
//   "address": "8.8.4.4:1",
//   "connect": [
//     {
//       "address": "8.8.4.4:1",
//       "failure": "generic_timeout_error",
//       "operation": "connect",
//       "proto": "tcp",
//       "t": 10.005494583,
//       "started": 8.4833e-05,
//       "oddity": "tcp.connect.timeout"
//     }
//   ]
// }
// ```
//
// So, we clearly see from the value of `t` that our 60 seconds
// default timeout did not hit, because there is a lower watchdog
// timeout (10 s). We also see again how the oddity is more
// precise than just the error alone.
//
// Let us now use a very small timeout:
//
// ```bash
// go run -race ./internal/tutorial/measurex/chapter02 -address 8.8.4.4:1 -timeout 100ms
// ```
//
// To get this JSON:
//
// ```JSON
// {
//   "network": "tcp",
//   "address": "8.8.4.4:1",
//   "connect": [
//     {
//       "address": "8.8.4.4:1",
//       "failure": "generic_timeout_error",
//       "operation": "connect",
//       "proto": "tcp",
//       "t": 0.10148025,
//       "started": 0.000122375,
//       "oddity": "tcp.connect.timeout"
//     }
//   ]
// }
// ```
//
// We see a timeout after ~0.1s. We enforce a reasonably small
// timeout for connecting, equal to 10 s, because we want to
// guarantee that measurements eventually terminate. Also, since
// often censorship is implemented by timing out, we don't want
// to spend to much time waiting for a timeout to expire.
//
// ## Conclusions
//
// We have seen how to measure the operation of connecting
// to a specific TCP endpoint.
//
// -=-=- StopHere -=-=-
