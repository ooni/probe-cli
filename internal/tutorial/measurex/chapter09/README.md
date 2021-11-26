
# Chapter IX: Parallel HTTPEndpoint measurements

The program we see here is _really_ similar to the one we
discussed in the previous chapter. The main difference
is the following: rather than looping through the list of
HTTPEndpoint, we call a function that runs through the
list of endpoints using a small pool of background workers.

There is a trade off between quick measurements and
false positives. A timeout is one of the most common
ways of censoring HTTPS and HTTP3 endpoints. So, if
we run measurements sequentially, a whole scan could
in principle take a long time. On the other hand,
if we run too many parallel measurements, we may cause
our own congestion and maybe some measurements will
fail because of that. Our solution to this problem is
to have low parallelism: at the moment of writing
this note, we have three workers. If you submit
more than three HTTPEndpoint at a a time, we will
service the first three immediately and all the
other endpoints will be queued for later measurement.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter09/main.go`.)

## main.go

The beginning of the program is pretty much the same.

```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type measurement struct {
	DNS       []*measurex.DNSMeasurement
	Endpoints []*measurex.HTTPEndpointMeasurement
}

func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

func main() {
	URL := flag.String("url", "https://blog.cloudflare.com/", "URL to fetch")
	address := flag.String("address", "8.8.4.4:53", "DNS-over-UDP server address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := &measurement{}
	m.DNS = append(m.DNS, mx.LookupHostUDP(ctx, parsed.Hostname(), *address))
	m.DNS = append(m.DNS, mx.LookupHTTPSSvcUDP(ctx, parsed.Hostname(), *address))
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
	httpEndpoints, err := measurex.AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
```

This is where the program changes. First, we need to create a jar
for cookies because the API we're about to call requires a
cookie jar. (We mostly use this API with redirects and we want
to have cookies with redirects because a small portion of the
URLs we typically test require cookies to properly redirect,
see https://github.com/ooni/probe/issues/1727 for more information).

Then, we call `HTTPEndpointGetParallel`. The arguments are:

- as usual, the context

- the cookie jar

- all the endpoints to measure

The parallelism argument tells the code how many parallel goroutines
to use for parallelizable operations. If this value is zero or negative,
the code will use a reasonably small default.

```Go
	cookies := measurex.NewCookieJar()
	const parallelism = 3
	for epnt := range mx.HTTPEndpointGetParallel(ctx, parallelism, cookies, httpEndpoints...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
```

The `HTTPEndpointGetParallel` method returns a channel where it
posts `HTTPEndpointMeasurements`. Once the input list has been
fully measured, this method closes the returned channel.

Like we did before, we append the resulting measurements to
our `m` container and we print it.

Exercise: here we're not using the OONI data format and we're
instead printing the internally used data structures. Can
you modify the code to emit data using OONI's data format here?
(Hint: there are conversion functions in `measurex`.)

```Go
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter09 | jq
```

Take a look at the JSON output. Can you spot that
endpoints measurements are run in parallel?

## Conclusion

We have seen how to run HTTPEndpoint measurements in parallel.

