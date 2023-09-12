
# Chapter IX: Parallel DNS lookups

The program we see here is _really_ similar to the one we
discussed in the previous chapter. The main difference
is the following: rather than performing DNS lookups
sequentially, we call a function that runs through the
list of resolvers and run them in parallel.

Again, we are going to use low parallelism for the same
rationale mentioned in chapter09.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter10/main.go`.)

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

	"github.com/ooni/probe-cli/v3/internal/legacy/measurex"
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
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := &measurement{}
```

The bulk of the difference is here. We create
a list of DNS resolvers. For each of them, we specify
the type and the endpoint address. (There is no
endpoint address for the system resolver, therefore
we leave its address empty.)

```Go
	resolvers := []*measurex.ResolverInfo{{
		Network: measurex.ResolverUDP,
		Address: "8.8.8.8:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "8.8.4.4:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "1.1.1.1:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "1.0.0.1:53",
	}, {
		Network: measurex.ResolverSystem,
		Address: "",
	}}
```

Then we call `LookupURLHostParallel`. This function runs
the queries that make sense given the input URL using a
pool of (currently three) background goroutines.

When I say "queries that make sense", I mostly mean
that we only query for HTTPSSvc when the input URL
scheme is "https". Otherwise, if it's just "http", it
does not make sense to send this query.

```Go
	const parallelism = 3
	for dns := range mx.LookupURLHostParallel(ctx, parallelism, parsed, resolvers...) {
		m.DNS = append(m.DNS, dns)
	}
```

The rest of the program is exactly like in chapter09.

```Go
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
	httpEndpoints, err := measurex.AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
	cookies := measurex.NewCookieJar()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, parallelism, cookies, httpEndpoints...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter10 | jq
```

Take a look at the JSON output. Can you spot that
DNS queries are run in parallel?

## Conclusion

We have seen how to run parallel DNS queries.

