
# Chapter VII: HTTPSSvc DNS queries

The program we see here is _really_ similar to the one we
discussed in the previous chapter. The main difference
is the following: now we also issue HTTPSSvc DNS queries
to discover HTTP/3 endpoints. (Because HTTPSSvc is
still a draft and is mostly implemented by Cloudflare
at this point, we are going to use as the example
input URL a Cloudflare URL.)

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter08/main.go`.)

## main.go

The beginning of the program is pretty much the same. We
have just amended our `measurement` type to contain multiple
`DNSMeasurement` results.

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
	address := flag.String("address", "8.8.4.4:53", "DNS-over-UDP server address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := &measurement{}
```
### Call LookupHTTPSSvc

Here we perform the `LookupHostUDP` we used in the
previous chapter and then we call `LookupHTTPSvcUDP`.

```Go
	m.DNS = append(m.DNS, mx.LookupHostUDP(ctx, parsed.Hostname(), *address))
	m.DNS = append(m.DNS, mx.LookupHTTPSSvcUDP(ctx, parsed.Hostname(), *address))
```

The `LookupHTTPSSvcUDP` function has the same signature
as `LookupHostUDP` _but_ it behaves differently. Rather than
querying for `A` and `AAAA`, it performs an `HTTPS` DNS
lookup. This query returns:

1. a list of ALPNs for the domain;

2. a list of IPv4 addresses;

3. a list of IPv6 addresses.

### Build an []HTTPEndpoint and run serial measurements

Here we call `AllHTTPEndpointsForURL` like we did in the
previous chapter. However, note that we pass it the
whole content of `m.DNS`, which now contains not only the
A/AAAA lookups results but also the HTTPS lookup results.

The `AllHTTPEndpointsForURL` function will recognize that
we also have HTTPS lookups and, if the "h3" ALPN is
present, will _also_ build HTTP/3 endpoints using "udp"
as the `HTTPEndpoint.Network`.

```Go
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
	httpEndpoints, err := measurex.AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
```

This is it. The rest of the program is exactly the same.

```Go
	for _, epnt := range httpEndpoints {
		m.Endpoints = append(m.Endpoints, mx.HTTPEndpointGetWithoutCookies(ctx, epnt))
	}
```

(Note that here, like in the previous chapter, we are not converting
to the OONI data format. Rather, we're just dumping the internally
used data structures. Exercise: can you modify this program to emit
a JSON compliant with the OONI data format by using the proper]
conversion functions exported by `measurex`?)

```Go
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter08 | jq
```

Please, check the JSON output. Do you recognize the fields
we have described in previous chapters? You should see
that, compared to previous chapters, now we're also testing
QUIC/HTTP3 endpoints.

Can you provoke common errors such as DNS resolution
errors, TCP connect errors, TLS handshake errors, and
HTTP round trip errors? What is a good way to cause
timeout and SNI mismatch errors for QUIC?

## Conclusion

We have seen how to extend fetching all the HTTPS
endpoints to include the QUIC/HTTP3 endpoints discovered
using HTTPSSvc.

