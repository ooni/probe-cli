
# Chapter VI: Getting a webpage from an HTTPS endpoint.

This chapter describes measuring getting a webpage from an
HTTPS endpoint.

Without further ado, let's describe our example `main.go` program
and let's use it to better understand this flow.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measure/chapter06/main.go`.)

## main.go

The initial part of the program is pretty much the same as the one
used in previous chapters, expect that we have a few more command line
flags now, so I will not add further comments.

```Go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/measure"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	sni := flag.String("sni", "dns.google", "SNI to use")
	urlPath := flag.String("url-path", "/", "URL path to use")
	hostHeader := flag.String("host-header", "dns.google", "Host header to use")
	timeout := flag.Duration("timeout", 10*time.Second, "timeout to use")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	URL := &url.URL{Scheme: "https", Host: *address, Path: *urlPath}
	begin := time.Now()
	trace := measure.NewTrace(begin)
	mx := measure.NewMeasurerStdlib(begin, log.Log, trace)
```

In previous examples, we have always provided the TLS config
inline. Here, we create a named variable to reduce the amount
of information packed in a single line of code. Apart from
that, this configuration is the same we have been providing
previously when handshaking to TLS endpoints on port 443.

```Go
	tlsConfig := &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
	}
```

The following is a new piece of code we have not
encountered so far. It creates a new jar for cookies
that prevents a domain from setting a cookie for an
unrelated domain. We need to keep track of cookies
when measuring because, among other things, some
redirections do not work without cookies.

See https://github.com/ooni/probe/issues/1727 for
more information on the behavior of URLS belonging
to the github.com/citizenlab/test-list repo, that
is, the URLs we most frequently test.

```Go
	cookies := measure.NewCookieJar()
```

The next step is creating an `HTTPRequest`. We use
a factory to create it that also forces using a specific
host header rather than using the URL's hostname.

```Go
	httpRequest := measure.NewHTTPRequestWithHostOverride(URL, cookies, *hostHeader)
```

We are now ready to run the `HTTPSEndpointGet` flow. The
arguments are:

- a context to carry timeout information;

- the address of the TCP endpoint to connect to;

- the TLS config to use for the handshake;

- the httpRequest struct that contains information regarding
sending the request and getting back a response.

```Go
	m := mx.HTTPSEndpointGet(ctx, *address, tlsConfig, httpRequest)
```

The rest of the program is pretty standard, so we are
not going to comment it in detail.

```Go
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## Running the example program

As before, let us start off with a vanilla run:

```bash
go run ./internal/tutorial/measure/chapter06
```

This is the JSON output. Let us comment it in detail:

```Javascript
{
  // This is the usual tcp_connect section
  "tcp_connect": {
    "network": "tcp",
    "address": "8.8.4.4:443",
    "started": 6299193,
    "completed": 28836661,
    "failure": null
  },

  // This is the usual tls_handshake section
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
    "started": 28843185,
    "completed": 69356361,
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

  // This is the usual network_events section
  "network_events": [
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 29197408,
      "completed": 29244947,
      "failure": null,
      "num_bytes": 280
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 29255399,
      "completed": 66999792,
      "failure": null,
      "num_bytes": 517
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 67345476,
      "completed": 67358273,
      "failure": null,
      "num_bytes": 3737
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 67359947,
      "completed": 67613648,
      "failure": null,
      "num_bytes": 565
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 69249176,
      "completed": 69281614,
      "failure": null,
      "num_bytes": 64
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 69709732,
      "completed": 69745166,
      "failure": null,
      "num_bytes": 86
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 69849876,
      "completed": 69859616,
      "failure": null,
      "num_bytes": 201
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 69929102,
      "completed": 90179507,
      "failure": null,
      "num_bytes": 62
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 90828517,
      "completed": 90863400,
      "failure": null,
      "num_bytes": 31
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 90876638,
      "completed": 90884695,
      "failure": null,
      "num_bytes": 31
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 90894094,
      "completed": 337577057,
      "failure": null,
      "num_bytes": 1990
    },
    {
      "operation": "read",
      "address": "8.8.4.4:443",
      "started": 338174299,
      "completed": 338182175,
      "failure": null,
      "num_bytes": 39
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 338192948,
      "completed": 338223742,
      "failure": null,
      "num_bytes": 39
    },
    {
      "operation": "write",
      "address": "8.8.4.4:443",
      "started": 338498478,
      "completed": 338509381,
      "failure": null,
      "num_bytes": 24
    }
  ],

  // This section is new and describes the HTTP exchange
  "http": {

    // This describes the request fields.
    "request": {
      "url": "https://8.8.4.4:443/",
      "host": "dns.google",
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

    // These are the usual timing and failure information.
    "started": 69414483,
    "completed": 338371121,
    "failure": null,

    // This describes the response: status code, the headers,
    // and the body (or a snapshot of the body)
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
          "object-src 'none';base-uri 'self';script-src 'nonce-3leAlnpPujKqFc3sBoXhTg==' 'strict-dynamic' 'report-sample' 'unsafe-eval' 'unsafe-inline' https: http:;report-uri https://csp.withgoogle.com/csp/honest_dns/1_0;frame-ancestors 'none'"
        ],
        "Content-Type": [
          "text/html; charset=UTF-8"
        ],
        "Date": [
          "Wed, 15 Sep 2021 14:41:05 GMT"
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
      "body": "PCFET0NUWVBFIGh0bWw+CjxodG1sIGxhbmc9ImVuIj4gPGhlYWQ+IDx0aXRsZT5Hb29nbGUgUHVibGljIEROUzwvdGl0bGU+ICA8bWV0YSBjaGFyc2V0PSJVVEYtOCI+IDxsaW5rIGhyZWY9Ii9zdGF0aWMvOTNkZDU5NTQvZmF2aWNvbi5wbmciIHJlbD0ic2hvcnRjdXQgaWNvbiIgdHlwZT0iaW1hZ2UvcG5nIj4gPGxpbmsgaHJlZj0iL3N0YXRpYy84MzZhZWJjNi9tYXR0ZXIubWluLmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPGxpbmsgaHJlZj0iL3N0YXRpYy9iODUzNmMzNy9zaGFyZWQuY3NzIiByZWw9InN0eWxlc2hlZXQiPiA8bWV0YSBuYW1lPSJ2aWV3cG9ydCIgY29udGVudD0id2lkdGg9ZGV2aWNlLXdpZHRoLCBpbml0aWFsLXNjYWxlPTEiPiAgPGxpbmsgaHJlZj0iL3N0YXRpYy9kMDVjZDZiYS9yb290LmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPC9oZWFkPiA8Ym9keT4gPHNwYW4gY2xhc3M9ImZpbGxlciB0b3AiPjwvc3Bhbj4gICA8ZGl2IGNsYXNzPSJsb2dvIiB0aXRsZT0iR29vZ2xlIFB1YmxpYyBETlMiPiA8ZGl2IGNsYXNzPSJsb2dvLXRleHQiPjxzcGFuPlB1YmxpYyBETlM8L3NwYW4+PC9kaXY+IDwvZGl2PiAgPGZvcm0gYWN0aW9uPSIvcXVlcnkiIG1ldGhvZD0iR0VUIj4gIDxkaXYgY2xhc3M9InJvdyI+IDxsYWJlbCBjbGFzcz0ibWF0dGVyLXRleHRmaWVsZC1vdXRsaW5lZCI+IDxpbnB1dCB0eXBlPSJ0ZXh0IiBuYW1lPSJuYW1lIiBwbGFjZWhvbGRlcj0iJm5ic3A7Ij4gPHNwYW4+RE5TIE5hbWU8L3NwYW4+IDxwIGNsYXNzPSJoZWxwIj4gRW50ZXIgYSBkb21haW4gKGxpa2UgZXhhbXBsZS5jb20pIG9yIElQIGFkZHJlc3MgKGxpa2UgOC44LjguOCBvciAyMDAxOjQ4NjA6NDg2MDo6ODg0NCkgaGVyZS4gPC9wPiA8L2xhYmVsPiA8YnV0dG9uIGNsYXNzPSJtYXR0ZXItYnV0dG9uLWNvbnRhaW5lZCBtYXR0ZXItcHJpbWFyeSIgdHlwZT0ic3VibWl0Ij5SZXNvbHZlPC9idXR0b24+IDwvZGl2PiA8L2Zvcm0+ICA8c3BhbiBjbGFzcz0iZmlsbGVyIGJvdHRvbSI+PC9zcGFuPiA8Zm9vdGVyIGNsYXNzPSJyb3ciPiA8YSBocmVmPSJodHRwczovL2RldmVsb3BlcnMuZ29vZ2xlLmNvbS9zcGVlZC9wdWJsaWMtZG5zIj5IZWxwPC9hPiA8YSBocmVmPSIvY2FjaGUiPkNhY2hlIEZsdXNoPC9hPiA8c3BhbiBjbGFzcz0iZmlsbGVyIj48L3NwYW4+IDxhIGhyZWY9Imh0dHBzOi8vZGV2ZWxvcGVycy5nb29nbGUuY29tL3NwZWVkL3B1YmxpYy1kbnMvZG9jcy91c2luZyI+IEdldCBTdGFydGVkIHdpdGggR29vZ2xlIFB1YmxpYyBETlMgPC9hPiA8L2Zvb3Rlcj4gICA8c2NyaXB0IG5vbmNlPSIzbGVBbG5wUHVqS3FGYzNzQm9YaFRnPT0iPmRvY3VtZW50LmZvcm1zWzBdLm5hbWUuZm9jdXMoKTs8L3NjcmlwdD4gPC9ib2R5PiA8L2h0bWw+"
    }
  }
}
```

A future version of this document will describe what
happens under several error conditions.

## Conclusion

We have seen how to measure the flow of fetching a
specific webpage from an HTTPS endpoint.

