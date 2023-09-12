
# Chapter VII: Measuring all the HTTPEndpoints for a domain

We are now going to combine DNS resolutions with getting
HTTPEndpoints. Conceptually, the DNS resolution yields
us a list of IP addresses. For each address, we build the
HTTPEndpoint and fetch it like we did in chapter06.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter07/main.go`.)

## main.go

We have package declaration and imports as usual.

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

```

Here we define an helper type for containing the DNS
measurement and the subsequent endpoints measurements.

```Go
type measurement struct {
	DNS       *measurex.DNSMeasurement
	Endpoints []*measurex.HTTPEndpointMeasurement
}

```

The rest of the program is quite similar to what we had before.

```Go
func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

func main() {
	URL := flag.String("url", "https://google.com/", "URL to fetch")
	address := flag.String("address", "8.8.4.4:53", "DNS-over-UDP server address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
```

This is where the main.go file starts to diverge. We create an
instance of our measurement type to hold the results.

```Go
	m := &measurement{}
```

Then we perform a DNS lookup using UDP like we saw in chapter03.

```Go
	m.DNS = mx.LookupHostUDP(ctx, parsed.Hostname(), *address)
```

Like we did in the previous chapter, we create suitable HTTP
headers for performing an HTTP measurement.

```Go
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
```

The following is an entirely new function we're learning
about just now. `AllHTTPEndpointsForURL` is a free function
in `measurex` that given:

- an already parsed HTTP/HTTPS URL

- headers we want to use

- the result of one or more DNS queries

builds us a list of HTTPEndpoint data structures.

```Go
	httpEndpoints, err := measurex.AllHTTPEndpointsForURL(parsed, headers, m.DNS)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
```

This function may fail if, for example, the URL is not HTTP/HTTPS. We
handle the error panicking, because this is an example program.

We are almost done now: we loop over all the endpoints and apply the
`HTTPEndpointGetWithoutCookies` method we have seen in chapter06.

```Go
	for _, epnt := range httpEndpoints {
		m.Endpoints = append(m.Endpoints, mx.HTTPEndpointGetWithoutCookies(ctx, epnt))
	}
```

Finally, we print the results. (Note that here we are not
converting to the OONI archival data format.)

```Go
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter07 | jq
```

Please, check the JSON output. Do you recognize the fields
we have described in previous chapters, even though we didn't
convert to the OONI data format? Can you modify the code to
use the OONI data format in the output by calling the proper
conversion functions exported by `measurex`?

Can you provoke common errors such as DNS resolution
errors, TCP connect errors, TLS handshake errors, and
HTTP round trip errors? How does the JSON change?

## Conclusion

We have seen how to combine DNS resolutions (chapter01 and
chapter03) with HTTPEndpoint GET (chapter06) to measure
all the HTTP endpoints for a given domain.

