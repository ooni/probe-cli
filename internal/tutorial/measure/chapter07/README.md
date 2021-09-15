
# Chapter VII: Measuring a single URL

This chapter is a digression. Rather than introducing you to
new functionality, we are going to put together everything we
have learned so far to write a simple URL scanner.

Without further ado, let's describe our example `main.go` program
and let's use it to better understand this flow.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measure/chapter07/main.go`.)

## main.go

We have the declaration of the package and imports (as usual). However,
this program is more complex than previous programs, so we are going
to split it into separate functions. Let us start from the `main` function.

```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/apex/log"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/measure"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

```

### The main function

```Go
func main() {
```

The initial part of this program should look familiar. We are
defining options using the `flag` package. Then we parse the
command line flags using `Parse`. Then we create a context attached
to a timeout for the whole scanning operation.

```Go
	URL := flag.String("url", "https://dns.google/", "URL to measure")
	timeout := flag.Duration("timeout", 30*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

Next we record the beginning of time, we create a new `Trace`,
and we make an instance of the `urlMeasurer`. This type
represents the possibility of measuring an URL (without any
redirection for now).

When we initialize the `urlMeasurer`, we pass to it an
instance of `Measurer` created like we did in previous chapters.

```Go
	begin := time.Now()
	trace := measure.NewTrace(begin)
	ux := &urlMeasurer{
		Begin:    begin,
		Logger:   log.Log,
		Measurer: measure.NewMeasurerStdlib(begin, log.Log, trace),
	}
```

We then call the `Measure` method of the `urlMeasurer`
passing to it the context (for timeouts) and the URL to
measure (without which we cannot measure :-).

```
	m := ux.Measure(ctx, *URL)
```

Finally, the usual code for printing the result.

```Go
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

### Data types

The `urlMeasurer` itself is pretty simple. It contains a
reference to the beginning of the measurement, to the logger,
and to a `Measurer` instance.

```Go

type urlMeasurer struct {
	Begin    time.Time
	Logger   measure.Logger
	Measurer *measure.Measurer
}

```

The `urlMeasurement` struct is the result of measuring an URL. This
type consists of the following subtypes:

- URLParse is a data structure we encounter now for the first
time. It represent the result of parsing an URL and also contains
useful helper fields to more easily generate TCP/UDP endpoints
once we have resolved a domain to IP addresses (e.g., the correct
port, which is either explicit in the URL or implicit from the scheme).

- DNS is a list of all the DNS resolutions we performed. We are
going to resolve the domain name in the URL using both the system
resolver and the `1.1.1.1:53` resolver.

- Endpoints contains the measurement of each endpoint discovered
through the DNS. (The program we have written in the previous
chapter was filling one of the structs saved in Endpoints.)

```Go

type urlMeasurement struct {
	URLParse  *measure.ParseURLResult          `json:"url_parse"`
	DNS       []*measure.LookupHostResult      `json:"dns"`
	Endpoints []*measure.HTTPEndpointGetResult `json:"endpoints"`
}

```

### The Measure method

The `Measure` method is the entry point of `urlMeasurer`. We have
already seen how it is invoked when we discussed `main`.

```Go

func (ux *urlMeasurer) Measure(ctx context.Context, URL string) *urlMeasurement {
```

The first step is parsing the input URL. To this end we use the
`Measurer.ParseURL` method. If this operation has not been successful,
then we stop here. The `urlMeasurement` will only contain a result
for parsing the URL and that will contain a failure.

```Go
	m := &urlMeasurement{}
	m.URLParse = ux.Measurer.ParseURL(URL)
	if !m.URLParse.Successful() {
		return m
	}
```

Now we extract the hostname (or IP address) from the URL.

```Go
	host := m.URLParse.Hostname()
```

We use this hostname to run the following flows:

1. resolving the hostname using the system resolver. We append
the result to the list of DNS resolutions.

```Go
	m.DNS = append(m.DNS, ux.Measurer.LookupHostSystem(ctx, host))
```

2. querying Cloudflare's public DNS-over-UDP endpoint 1.1.1.1:53
to resolve the hostname with both A (IPv4) and AAAA (IPv6). We
append also these results to the list of DNS resolutions.

```Go
	const cfDNS = "1.1.1.1:53"
	m.DNS = append(m.DNS, ux.Measurer.LookupHostUDP(ctx, host, dns.TypeA, cfDNS))
	m.DNS = append(m.DNS, ux.Measurer.LookupHostUDP(ctx, host, dns.TypeAAAA, cfDNS))
```

The next step is to reduce the results of all resolutions (which
only contain IP addresses) to a unique set of endpoints. That
is, we remove duplicate addresses and we correctly append
the port. (URL parsing will fail if there is no explicit port
and it cannot guess the port from the scheme, hence we can
trust `Port` here to be a valid port.)

Note that all the above DNS resolutions may have failed. This
is fine: `MergeEndpoints` will return an empty list if the
input list is also empty.

```Go
	epnts := ux.Measurer.MergeEndpoints(m.DNS, m.URLParse.Port)
```

As the final step, we cycle through the (possibly empty) list
of endpoints and we measure each of them. Note how measuring an
endpoint requires the context, the original parsed URL, and
the address of the endpoint itself.

After we measure each endpoint, we append the result to the
list of endpoint measurements.

```Go
	for _, epnt := range epnts {
		em := ux.measureEndpoint(ctx, m.URLParse.Parsed, epnt)
		m.Endpoints = append(m.Endpoints, em)
	}
```

When we are done, we return the measurement result to the caller.

```Go
	return m
}

```

### Measuring a single endpoint

```Go

func (ux *urlMeasurer) measureEndpoint(ctx context.Context,
	URL *url.URL, endpoint string) *measure.HTTPEndpointGetResult {
```

We create a cookie jar like we did before. Since we are not
asking the question of redirection for now, it does not matter
much creating this jar at another place.

```Go
	cookies := measure.NewCookieJar()
```

We create the TLS configuration. This time we use a factory
function for creating it. The false argument indicates we
are using HTTPS as opposed to HTTP3. (This factory is functionally
equivalent to the code we have been using thus far for
creating TLS configs in the previous chapters.)

```Go
	tlsConfig := measure.NewTLSConfigFromURL(URL, false)
```

We create an HTTP request like we did in the previous chapter.

```Go
	httpRequest := measure.NewHTTPRequest(URL, cookies)
```

Finally, we invoke the flow and return its result (like we
did in the previous chapter).

```Go
	return ux.Measurer.HTTPSEndpointGet(ctx, endpoint, tlsConfig, httpRequest)
}

```

## Running the example program

As before, let us start off with a vanilla run:

```bash
go run ./internal/tutorial/measure/chapter07
```

We obtain this JSON. Lets us comment it in detail:

```JavaScript
{

  // We start with information on URL parsing. All was good
  // and we deduced the port (443) from the scheme.
  "url_parse": {
    "url": "https://dns.google/",
    "failure": null,
    "port": "443"
  },

  // This is the list of DNS resolutions.
  "dns": [

    // The first entry here is the system resolver one. We have
    // seen this data format many times already.
    {
      "engine": "system",
      "domain": "dns.google",
      "started": 21285,
      "completed": 28371443,
      "failure": null,
      "addrs": [
        "8.8.8.8",
        "8.8.4.4",
        "2001:4860:4860::8888",
        "2001:4860:4860::8844"
      ]
    },

    // Now we have the UDP query to clouflare for A. We have
    // already seen this data format.
    {
      "engine": "udp",
      "address": "1.1.1.1:53",
      "query_type": "A",
      "domain": "dns.google",
      "started": 28507328,
      "completed": 49651298,
      "query": "3P0BAAABAAAAAAAAA2RucwZnb29nbGUAAAEAAQ==",
      "failure": null,
      "addrs": [
        "8.8.8.8",
        "8.8.4.4"
      ],
      "reply": "3P2BgAABAAIAAAAAA2RucwZnb29nbGUAAAEAAcAMAAEAAQAAAHkABAgICAjADAABAAEAAAB5AAQICAQE",
      "network_events": [
        {
          "operation": "write",
          "address": "1.1.1.1:53",
          "started": 28578878,
          "completed": 28634314,
          "failure": null,
          "num_bytes": 28
        },
        {
          "operation": "read",
          "address": "1.1.1.1:53",
          "started": 28651400,
          "completed": 49616816,
          "failure": null,
          "num_bytes": 60
        }
      ]
    },

    // Same as above but this is the AAAA query.
    {
      "engine": "udp",
      "address": "1.1.1.1:53",
      "query_type": "AAAA",
      "domain": "dns.google",
      "started": 50041006,
      "completed": 72281907,
      "query": "/tgBAAABAAAAAAAAA2RucwZnb29nbGUAABwAAQ==",
      "failure": null,
      "addrs": [
        "2001:4860:4860::8844",
        "2001:4860:4860::8888"
      ],
      "reply": "/tiBgAABAAIAAAAAA2RucwZnb29nbGUAABwAAcAMABwAAQAAAlIAECABSGBIYAAAAAAAAAAAiETADAAcAAEAAAJSABAgAUhgSGAAAAAAAAAAAIiI",
      "network_events": [
        {
          "operation": "write",
          "address": "1.1.1.1:53",
          "started": 50106172,
          "completed": 50148541,
          "failure": null,
          "num_bytes": 28
        },
        {
          "operation": "read",
          "address": "1.1.1.1:53",
          "started": 50161248,
          "completed": 72246753,
          "failure": null,
          "num_bytes": 84
        }
      ]
    }
  ],

  // Now we have all the endpoints. Each entry of this list
  // is a data structure like the one we saw in the previous chapter.
  "endpoints": [
    {
      "tcp_connect": {
        "network": "tcp",
        "address": "8.8.8.8:443",
        "started": 78219293,
        "completed": 101032034,
        "failure": null
      },
      "tls_handshake": {
        "engine": "stdlib",
        "address": "8.8.8.8:443",
        "config": {
          "sni": "dns.google",
          "alpn": [
            "h2",
            "http/1.1"
          ],
          "no_tls_verify": false
        },
        "started": 101038227,
        "completed": 142327481,
        "failure": null,
        "connection_state": {
          "tls_version": "TLSv1.3",
          "cipher_suite": "TLS_AES_128_GCM_SHA256",
          "negotiated_protocol": "h2",
          "peer_certificates": [
            "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"
          ]
        }
      },
      "network_events": [
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 102124195,
          "completed": 102161193,
          "failure": null,
          "num_bytes": 280
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 102202299,
          "completed": 139916473,
          "failure": null,
          "num_bytes": 517
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 140188033,
          "completed": 140194887,
          "failure": null,
          "num_bytes": 901
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 140196461,
          "completed": 140334701,
          "failure": null,
          "num_bytes": 2836
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 140336965,
          "completed": 140625080,
          "failure": null,
          "num_bytes": 564
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 142216859,
          "completed": 142255340,
          "failure": null,
          "num_bytes": 64
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 142589030,
          "completed": 142616097,
          "failure": null,
          "num_bytes": 86
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 142692307,
          "completed": 142699622,
          "failure": null,
          "num_bytes": 201
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 142727521,
          "completed": 163027861,
          "failure": null,
          "num_bytes": 62
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 163065089,
          "completed": 163097006,
          "failure": null,
          "num_bytes": 31
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 163106676,
          "completed": 163310283,
          "failure": null,
          "num_bytes": 31
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 163326277,
          "completed": 573845070,
          "failure": null,
          "num_bytes": 576
        },
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 574360141,
          "completed": 575348615,
          "failure": null,
          "num_bytes": 1453
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 575422289,
          "completed": 575453886,
          "failure": null,
          "num_bytes": 39
        },
        {
          "operation": "write",
          "address": "8.8.8.8:443",
          "started": 575545708,
          "completed": 575553455,
          "failure": null,
          "num_bytes": 24
        }
      ],
      "http": {
        "request": {
          "url": "https://dns.google/",
          "host": "",
          "headers": {
            "Accept": [
              "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
            ],
            "Accept-Language": [
              "en-US;q=0.8,en;q=0.5"
            ],
            "User-Agent": [
              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
            ]
          },
          "max_body_size": 8192
        },
        "started": 142369399,
        "completed": 574557936,
        "failure": null,
        "response": {
          "status_code": 200,
          "headers": {
            "Accept-Ranges": [
              "none"
            ],
            "Alt-Svc": [
              "h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000,h3-T051=\":443\"; ma=2592000,h3-Q050=\":443\"; ma=2592000,h3-Q046=\":443\"; ma=2592000,h3-Q043=\":443\"; ma=2592000,quic=\":443\"; ma=2592000; v=\"46,43\""
            ],
            "Cache-Control": [
              "private"
            ],
            "Content-Security-Policy": [
              "object-src 'none';base-uri 'self';script-src 'nonce-pP1rx2l3OmYdMApFIYG/NA==' 'strict-dynamic' 'report-sample' 'unsafe-eval' 'unsafe-inline' https: http:;report-uri https://csp.withgoogle.com/csp/honest_dns/1_0;frame-ancestors 'none'"
            ],
            "Content-Type": [
              "text/html; charset=UTF-8"
            ],
            "Date": [
              "Wed, 15 Sep 2021 15:27:55 GMT"
            ],
            "Server": [
              "scaffolding on HTTPServer2"
            ],
            "Strict-Transport-Security": [
              "max-age=31536000; includeSubDomains; preload"
            ],
            "Vary": [
              "Accept-Encoding"
            ],
            "X-Content-Type-Options": [
              "nosniff"
            ],
            "X-Frame-Options": [
              "SAMEORIGIN"
            ],
            "X-Xss-Protection": [
              "0"
            ]
          },
          "body": "PCFET0NUWVBFIGh0bWw+CjxodG1sIGxhbmc9ImVuIj4gPGhlYWQ+IDx0aXRsZT5Hb29nbGUgUHVibGljIEROUzwvdGl0bGU+ICA8bWV0YSBjaGFyc2V0PSJVVEYtOCI+IDxsaW5rIGhyZWY9Ii9zdGF0aWMvOTNkZDU5NTQvZmF2aWNvbi5wbmciIHJlbD0ic2hvcnRjdXQgaWNvbiIgdHlwZT0iaW1hZ2UvcG5nIj4gPGxpbmsgaHJlZj0iL3N0YXRpYy84MzZhZWJjNi9tYXR0ZXIubWluLmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPGxpbmsgaHJlZj0iL3N0YXRpYy9iODUzNmMzNy9zaGFyZWQuY3NzIiByZWw9InN0eWxlc2hlZXQiPiA8bWV0YSBuYW1lPSJ2aWV3cG9ydCIgY29udGVudD0id2lkdGg9ZGV2aWNlLXdpZHRoLCBpbml0aWFsLXNjYWxlPTEiPiAgPGxpbmsgaHJlZj0iL3N0YXRpYy9kMDVjZDZiYS9yb290LmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPC9oZWFkPiA8Ym9keT4gPHNwYW4gY2xhc3M9ImZpbGxlciB0b3AiPjwvc3Bhbj4gICA8ZGl2IGNsYXNzPSJsb2dvIiB0aXRsZT0iR29vZ2xlIFB1YmxpYyBETlMiPiA8ZGl2IGNsYXNzPSJsb2dvLXRleHQiPjxzcGFuPlB1YmxpYyBETlM8L3NwYW4+PC9kaXY+IDwvZGl2PiAgPGZvcm0gYWN0aW9uPSIvcXVlcnkiIG1ldGhvZD0iR0VUIj4gIDxkaXYgY2xhc3M9InJvdyI+IDxsYWJlbCBjbGFzcz0ibWF0dGVyLXRleHRmaWVsZC1vdXRsaW5lZCI+IDxpbnB1dCB0eXBlPSJ0ZXh0IiBuYW1lPSJuYW1lIiBwbGFjZWhvbGRlcj0iJm5ic3A7Ij4gPHNwYW4+RE5TIE5hbWU8L3NwYW4+IDxwIGNsYXNzPSJoZWxwIj4gRW50ZXIgYSBkb21haW4gKGxpa2UgZXhhbXBsZS5jb20pIG9yIElQIGFkZHJlc3MgKGxpa2UgOC44LjguOCBvciAyMDAxOjQ4NjA6NDg2MDo6ODg0NCkgaGVyZS4gPC9wPiA8L2xhYmVsPiA8YnV0dG9uIGNsYXNzPSJtYXR0ZXItYnV0dG9uLWNvbnRhaW5lZCBtYXR0ZXItcHJpbWFyeSIgdHlwZT0ic3VibWl0Ij5SZXNvbHZlPC9idXR0b24+IDwvZGl2PiA8L2Zvcm0+ICA8c3BhbiBjbGFzcz0iZmlsbGVyIGJvdHRvbSI+PC9zcGFuPiA8Zm9vdGVyIGNsYXNzPSJyb3ciPiA8YSBocmVmPSJodHRwczovL2RldmVsb3BlcnMuZ29vZ2xlLmNvbS9zcGVlZC9wdWJsaWMtZG5zIj5IZWxwPC9hPiA8YSBocmVmPSIvY2FjaGUiPkNhY2hlIEZsdXNoPC9hPiA8c3BhbiBjbGFzcz0iZmlsbGVyIj48L3NwYW4+IDxhIGhyZWY9Imh0dHBzOi8vZGV2ZWxvcGVycy5nb29nbGUuY29tL3NwZWVkL3B1YmxpYy1kbnMvZG9jcy91c2luZyI+IEdldCBTdGFydGVkIHdpdGggR29vZ2xlIFB1YmxpYyBETlMgPC9hPiA8L2Zvb3Rlcj4gICA8c2NyaXB0IG5vbmNlPSJwUDFyeDJsM09tWWRNQXBGSVlHL05BPT0iPmRvY3VtZW50LmZvcm1zWzBdLm5hbWUuZm9jdXMoKTs8L3NjcmlwdD4gPC9ib2R5PiA8L2h0bWw+"
        }
      }
    },
    {
      "tcp_connect": {
        "network": "tcp",
        "address": "8.8.4.4:443",
        "started": 582951395,
        "completed": 604500385,
        "failure": null
      },
      "tls_handshake": {
        "engine": "stdlib",
        "address": "8.8.4.4:443",
        "config": {
          "sni": "dns.google",
          "alpn": [
            "h2",
            "http/1.1"
          ],
          "no_tls_verify": false
        },
        "started": 604505836,
        "completed": 643354350,
        "failure": null,
        "connection_state": {
          "tls_version": "TLSv1.3",
          "cipher_suite": "TLS_AES_128_GCM_SHA256",
          "negotiated_protocol": "h2",
          "peer_certificates": [
            "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"
          ]
        }
      },
      "network_events": [
        {
          "operation": "read",
          "address": "8.8.8.8:443",
          "started": 575461722,
          "completed": 576037099,
          "failure": "unknown_failure: read tcp [scrubbed]->[scrubbed]: use of closed network connection",
          "num_bytes": 0
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 604838524,
          "completed": 604880923,
          "failure": null,
          "num_bytes": 280
        },
        {
          "operation": "read",
          "address": "8.8.4.4:443",
          "started": 604889541,
          "completed": 641018900,
          "failure": null,
          "num_bytes": 517
        },
        {
          "operation": "read",
          "address": "8.8.4.4:443",
          "started": 641288546,
          "completed": 641295501,
          "failure": null,
          "num_bytes": 2319
        },
        {
          "operation": "read",
          "address": "8.8.4.4:443",
          "started": 641296543,
          "completed": 641614901,
          "failure": null,
          "num_bytes": 1983
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 643260503,
          "completed": 643289754,
          "failure": null,
          "num_bytes": 64
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 643609155,
          "completed": 643643537,
          "failure": null,
          "num_bytes": 86
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 643743807,
          "completed": 643752495,
          "failure": null,
          "num_bytes": 201
        },
        {
          "operation": "read",
          "address": "8.8.4.4:443",
          "started": 643787018,
          "completed": 662805758,
          "failure": null,
          "num_bytes": 93
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 662845101,
          "completed": 662872769,
          "failure": null,
          "num_bytes": 31
        },
        {
          "operation": "read",
          "address": "8.8.4.4:443",
          "started": 662886438,
          "completed": 946120810,
          "failure": null,
          "num_bytes": 1990
        },
        {
          "operation": "write",
          "address": "8.8.4.4:443",
          "started": 946518525,
          "completed": 946551353,
          "failure": null,
          "num_bytes": 24
        }
      ],
      "http": {
        "request": {
          "url": "https://dns.google/",
          "host": "",
          "headers": {
            "Accept": [
              "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8"
            ],
            "Accept-Language": [
              "en-US;q=0.8,en;q=0.5"
            ],
            "User-Agent": [
              "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/90.0.4430.93 Safari/537.36"
            ]
          },
          "max_body_size": 8192
        },
        "started": 643379453,
        "completed": 946441683,
        "failure": null,
        "response": {
          "status_code": 200,
          "headers": {
            "Accept-Ranges": [
              "none"
            ],
            "Alt-Svc": [
              "h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000,h3-T051=\":443\"; ma=2592000,h3-Q050=\":443\"; ma=2592000,h3-Q046=\":443\"; ma=2592000,h3-Q043=\":443\"; ma=2592000,quic=\":443\"; ma=2592000; v=\"46,43\""
            ],
            "Cache-Control": [
              "private"
            ],
            "Content-Security-Policy": [
              "object-src 'none';base-uri 'self';script-src 'nonce-oda+891dNAx5r7+TNhixgw==' 'strict-dynamic' 'report-sample' 'unsafe-eval' 'unsafe-inline' https: http:;report-uri https://csp.withgoogle.com/csp/honest_dns/1_0;frame-ancestors 'none'"
            ],
            "Content-Type": [
              "text/html; charset=UTF-8"
            ],
            "Date": [
              "Wed, 15 Sep 2021 15:27:56 GMT"
            ],
            "Server": [
              "scaffolding on HTTPServer2"
            ],
            "Strict-Transport-Security": [
              "max-age=31536000; includeSubDomains; preload"
            ],
            "Vary": [
              "Accept-Encoding"
            ],
            "X-Content-Type-Options": [
              "nosniff"
            ],
            "X-Frame-Options": [
              "SAMEORIGIN"
            ],
            "X-Xss-Protection": [
              "0"
            ]
          },
          "body": "PCFET0NUWVBFIGh0bWw+CjxodG1sIGxhbmc9ImVuIj4gPGhlYWQ+IDx0aXRsZT5Hb29nbGUgUHVibGljIEROUzwvdGl0bGU+ICA8bWV0YSBjaGFyc2V0PSJVVEYtOCI+IDxsaW5rIGhyZWY9Ii9zdGF0aWMvOTNkZDU5NTQvZmF2aWNvbi5wbmciIHJlbD0ic2hvcnRjdXQgaWNvbiIgdHlwZT0iaW1hZ2UvcG5nIj4gPGxpbmsgaHJlZj0iL3N0YXRpYy84MzZhZWJjNi9tYXR0ZXIubWluLmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPGxpbmsgaHJlZj0iL3N0YXRpYy9iODUzNmMzNy9zaGFyZWQuY3NzIiByZWw9InN0eWxlc2hlZXQiPiA8bWV0YSBuYW1lPSJ2aWV3cG9ydCIgY29udGVudD0id2lkdGg9ZGV2aWNlLXdpZHRoLCBpbml0aWFsLXNjYWxlPTEiPiAgPGxpbmsgaHJlZj0iL3N0YXRpYy9kMDVjZDZiYS9yb290LmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPC9oZWFkPiA8Ym9keT4gPHNwYW4gY2xhc3M9ImZpbGxlciB0b3AiPjwvc3Bhbj4gICA8ZGl2IGNsYXNzPSJsb2dvIiB0aXRsZT0iR29vZ2xlIFB1YmxpYyBETlMiPiA8ZGl2IGNsYXNzPSJsb2dvLXRleHQiPjxzcGFuPlB1YmxpYyBETlM8L3NwYW4+PC9kaXY+IDwvZGl2PiAgPGZvcm0gYWN0aW9uPSIvcXVlcnkiIG1ldGhvZD0iR0VUIj4gIDxkaXYgY2xhc3M9InJvdyI+IDxsYWJlbCBjbGFzcz0ibWF0dGVyLXRleHRmaWVsZC1vdXRsaW5lZCI+IDxpbnB1dCB0eXBlPSJ0ZXh0IiBuYW1lPSJuYW1lIiBwbGFjZWhvbGRlcj0iJm5ic3A7Ij4gPHNwYW4+RE5TIE5hbWU8L3NwYW4+IDxwIGNsYXNzPSJoZWxwIj4gRW50ZXIgYSBkb21haW4gKGxpa2UgZXhhbXBsZS5jb20pIG9yIElQIGFkZHJlc3MgKGxpa2UgOC44LjguOCBvciAyMDAxOjQ4NjA6NDg2MDo6ODg0NCkgaGVyZS4gPC9wPiA8L2xhYmVsPiA8YnV0dG9uIGNsYXNzPSJtYXR0ZXItYnV0dG9uLWNvbnRhaW5lZCBtYXR0ZXItcHJpbWFyeSIgdHlwZT0ic3VibWl0Ij5SZXNvbHZlPC9idXR0b24+IDwvZGl2PiA8L2Zvcm0+ICA8c3BhbiBjbGFzcz0iZmlsbGVyIGJvdHRvbSI+PC9zcGFuPiA8Zm9vdGVyIGNsYXNzPSJyb3ciPiA8YSBocmVmPSJodHRwczovL2RldmVsb3BlcnMuZ29vZ2xlLmNvbS9zcGVlZC9wdWJsaWMtZG5zIj5IZWxwPC9hPiA8YSBocmVmPSIvY2FjaGUiPkNhY2hlIEZsdXNoPC9hPiA8c3BhbiBjbGFzcz0iZmlsbGVyIj48L3NwYW4+IDxhIGhyZWY9Imh0dHBzOi8vZGV2ZWxvcGVycy5nb29nbGUuY29tL3NwZWVkL3B1YmxpYy1kbnMvZG9jcy91c2luZyI+IEdldCBTdGFydGVkIHdpdGggR29vZ2xlIFB1YmxpYyBETlMgPC9hPiA8L2Zvb3Rlcj4gICA8c2NyaXB0IG5vbmNlPSJvZGErODkxZE5BeDVyNytUTmhpeGd3PT0iPmRvY3VtZW50LmZvcm1zWzBdLm5hbWUuZm9jdXMoKTs8L3NjcmlwdD4gPC9ib2R5PiA8L2h0bWw+"
        }
      }
    },

    // The last two entries have a simple explanation: I do not
    // have support for IPv6 with Vodafone Italia.
    {
      "tcp_connect": {
        "network": "tcp",
        "address": "[2001:4860:4860::8888]:443",
        "started": 954660947,
        "completed": 954877021,
        "failure": "network_unreachable"
      },
      "network_events": null,
      "http": null
    },
    {
      "tcp_connect": {
        "network": "tcp",
        "address": "[2001:4860:4860::8844]:443",
        "started": 961186788,
        "completed": 961389393,
        "failure": "network_unreachable"
      },
      "network_events": null,
      "http": null
    }
  ]
}
```

A future version of this document will describe what
happens under several error conditions.

## Conclusion

We have seen how we can combine what we have learned so
far to measure a single URL. No redirections for now. We're
going to look into redirections in a future chapter.

