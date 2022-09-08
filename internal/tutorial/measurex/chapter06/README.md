
# Chapter VI: Getting a webpage from an HTTP/HTTPS/HTTP3 endpoint.

This chapter describes measuring the retrieval of a webpage from an
HTTPS endpoint. We have seen how to TCP connect, we have
seen how to TLS handshake, now it's time to see how we can
combine these operations with that of fetching a webpage from a
given TCP endpoint speaking HTTP and TLS. (As well as
providing you with information on how to otherwise fetch
from HTTP and HTTP/3 endpoints.)

The program we're going to write, `main.go`, will use a
high-level operation to perform this measurement in a
single API call. The code implementing this API call will
combine the operations we have seen in previous chapter
with the "give me the webpage" operation. We are still
quite far away from the ability of "measuring a URL" but
we are increasingly moving towards more complex operations.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter06/main.go`.)

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

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

```

We have factored the three lines to print a measurement
into the following utility function called `print`.

```Go
func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

The initial part of the program is pretty much the same as the one
used in previous chapters, expect that we have a few more command line
flags now, so I will not add further comments.

```Go
func main() {
	sni := flag.String("sni", "dns.google", "value for SNI extension")
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
```

### HTTPEndpoint: a description of what to measure

First of all, let us create a description of the endpoint
for `measurex`. Up to this point we have used endpoint
to describe a level-4-like address. Therefore, we have seen
TCP endpoints being used for the TCP connect and the TLS
handshake, and we have seen QUIC endpoints being used
for the QUIC handshake. Now, however, we are going
to need more information to characterize the endpoint.

```Go
	epnt := &measurex.HTTPEndpoint{
		Domain:  *sni,
		Network: "tcp",
		Address: *address,
		SNI:     *sni,
		ALPN:    []string{"h2", "http/1.1"},
		URL: &url.URL{
			Scheme: "https",
			Host:   *sni,
			Path:   "/",
		},
		Header: measurex.NewHTTPRequestHeaderForMeasuring(),
	}
```

In fact, in the above definition we recognize fields
we have already discussed, such as:

- `Network`, describing whether to use "tcp" or "udp";

- `Address`, which is the endpoint address.

But we also need to combine into this view of the
endpoint additional fields for TLS:

- `SNI`, to set the SNI;

- `ALPN`, to set the ALPN;

But then we also need to specify:

- the URL to use;

- the headers to use (for which we're using a handy
factory for creating reasonable defaults for measuring).

This API is not the highest level API with which to do
the job, but it's still handy to introduce the
`measurex.HTTPEndpoint` data structure since it's
used by higher level APIs.

(You may also be wondering about the CA pool. It turns
out that for APIs such as this one and for higher
level APIs, the default is to always use the bundled
Mozilla CA pool, because this is what we use in
most cases for performing measurements.)

### HTTPEndpointGetWithoutCookies

When used with an HTTP URL, the `HTTPEndpointGetWithoutCookies`
method combines two operations:

- TCP connect

- HTTP GET

When the URL is HTTPS, we do:

- TCP connect

- TLS handshake

- HTTP GET (or HTTP/2 GET depending on the ALPN)

When the `HTTPEndpoint.Network` field value
is QUIC, instead we do:

- QUIC handshake

- HTTP/3 GET

```Go
	m := mx.HTTPEndpointGetWithoutCookies(ctx, epnt)
```

The arguments for `HTTPEndpointGetWithDBWithoutCookies` are:

- the context for deadline/timeout

- the HTTPEndpoint descriptor

The result is an `HTTPEndpointMeasurement` which
you can inspect with

```
go doc ./internal/measurex.HTTPEndpointMeasurement
```

### Printing the measurement

Let us now print the resulting measurement.

```Go
	print(measurex.NewArchivalHTTPEndpointMeasurement(m))
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter06 | jq
```

This is the JSON output. Let us comment it in detail:

```Javascript
{
  // The returned type is called HTTPEndpointMeasurement
  // and you see that here on top we indeed have the
  // information on the endpoint and the URL.
  "url": "https://dns.google/",
  "network": "tcp",
  "address": "8.8.4.4:443",

  // These are the I/O operations we have already seen
  // in previous chapters
  "network_events": [
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 280,
      "operation": "write",
      "proto": "tcp",
      "t": 0.045800292,
      "started": 0.045782167,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 517,
      "operation": "read",
      "proto": "tcp",
      "t": 0.082571,
      "started": 0.045805458,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 4303,
      "operation": "read",
      "proto": "tcp",
      "t": 0.084400542,
      "started": 0.084372667,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 64,
      "operation": "write",
      "proto": "tcp",
      "t": 0.086762625,
      "started": 0.086748292,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 86,
      "operation": "write",
      "proto": "tcp",
      "t": 0.087851,
      "started": 0.087837625,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 201,
      "operation": "write",
      "proto": "tcp",
      "t": 0.089527292,
      "started": 0.089507958,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 93,
      "operation": "read",
      "proto": "tcp",
      "t": 0.168585625,
      "started": 0.088068375,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 31,
      "operation": "write",
      "proto": "tcp",
      "t": 0.168713542,
      "started": 0.168671417,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 2000,
      "operation": "read",
      "proto": "tcp",
      "t": 0.468671417,
      "started": 0.168759333,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 39,
      "operation": "read",
      "proto": "tcp",
      "t": 0.47118175,
      "started": 0.471169667,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 39,
      "operation": "write",
      "proto": "tcp",
      "t": 0.471335458,
      "started": 0.471268583,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 24,
      "operation": "write",
      "proto": "tcp",
      "t": 0.471865,
      "started": 0.471836292,
      "oddity": ""
    }
  ],

  // Internally, HTTPEndpointGetWithoutCookies calls
  // TCPConnect and here we see the corresponding event
  "tcp_connect": [
    {
      "ip": "8.8.4.4",
      "port": 443,
      "t": 0.043644958,
      "status": {
        "blocked": false,
        "failure": null,
        "success": true
      },
      "started": 0.022849458,
      "oddity": ""
    }
  ],

  // Internally, HTTPEndpointGetWithoutCookies calls TLSConnectAndHandshake,
  // and here's the resulting handshake event. Of course, if we
  // specified a QUIC endpoint we would instead see here a
  // QUIC handshake event. And, we would not see any handshake
  // if the URL was instead an HTTP URL.
  "tls_handshakes": [
    {
      "cipher_suite": "TLS_AES_128_GCM_SHA256",
      "failure": null,
      "negotiated_proto": "h2",
      "tls_version": "TLSv1.3",
      "peer_certificates": [
        {
          "data": "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",
          "format": "base64"
        },
        {
          "data": "MIIFljCCA36gAwIBAgINAgO8U1lrNMcY9QFQZjANBgkqhkiG9w0BAQsFADBHMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzEUMBIGA1UEAxMLR1RTIFJvb3QgUjEwHhcNMjAwODEzMDAwMDQyWhcNMjcwOTMwMDAwMDQyWjBGMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzETMBEGA1UEAxMKR1RTIENBIDFDMzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPWI3+dijB43+DdCkH9sh9D7ZYIl/ejLa6T/belaI+KZ9hzpkgOZE3wJCor6QtZeViSqejOEH9Hpabu5dOxXTGZok3c3VVP+ORBNtzS7XyV3NzsXlOo85Z3VvMO0Q+sup0fvsEQRY9i0QYXdQTBIkxu/t/bgRQIh4JZCF8/ZK2VWNAcmBA2o/X3KLu/qSHw3TT8An4Pf73WELnlXXPxXbhqW//yMmqaZviXZf5YsBvcRKgKAgOtjGDxQSYflispfGStZloEAoPtR28p3CwvJlk/vcEnHXG0g/Zm0tOLKLnf9LdwLtmsTDIwZKxeWmLnwi/agJ7u2441Rj72ux5uxiZ0CAwEAAaOCAYAwggF8MA4GA1UdDwEB/wQEAwIBhjAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUinR/r4XN7pXNPZzQ4kYU83E1HScwHwYDVR0jBBgwFoAU5K8rJnEaK0gnhS9SZizv8IkTcT4waAYIKwYBBQUHAQEEXDBaMCYGCCsGAQUFBzABhhpodHRwOi8vb2NzcC5wa2kuZ29vZy9ndHNyMTAwBggrBgEFBQcwAoYkaHR0cDovL3BraS5nb29nL3JlcG8vY2VydHMvZ3RzcjEuZGVyMDQGA1UdHwQtMCswKaAnoCWGI2h0dHA6Ly9jcmwucGtpLmdvb2cvZ3RzcjEvZ3RzcjEuY3JsMFcGA1UdIARQME4wOAYKKwYBBAHWeQIFAzAqMCgGCCsGAQUFBwIBFhxodHRwczovL3BraS5nb29nL3JlcG9zaXRvcnkvMAgGBmeBDAECATAIBgZngQwBAgIwDQYJKoZIhvcNAQELBQADggIBAIl9rCBcDDy+mqhXlRu0rvqrpXJxtDaV/d9AEQNMwkYUuxQkq/BQcSLbrcRuf8/xam/IgxvYzolfh2yHuKkMo5uhYpSTld9brmYZCwKWnvy15xBpPnrLRklfRuFBsdeYTWU0AIAaP0+fbH9JAIFTQaSSIYKCGvGjRFsqUBITTcFTNvNCCK9U+o53UxtkOCcXCb1YyRt8OS1b887U7ZfbFAO/CVMkH8IMBHmYJvJh8VNS/UKMG2YrPxWhu//2m+OBmgEGcYk1KCTd4b3rGS3hSMs9WYNRtHTGnXzGsYZbr8w0xNPM1IERlQCh9BIiAfq0g3GvjLeMcySsN1PCAJA/Ef5c7TaUEDu9Ka7ixzpiO2xj2YC/WXGsYye5TBeg2vZzFb8q3o/zpWwygTMD0IZRcZk0upONXbVRWPeyk+gB9lm+cZv9TSjOz23HFtz30dZGm6fKa+l3D/2gthsjgx0QGtkJAITgRNOidSOzNIb2ILCkXhAd4FJGAJ2xDx8hcFH1mt0G/FX0Kw4zd8NLQsLxdxP8c4CU6x+7Nz/OAipmsHMdMqUybDKwjuDEI/9bfU1lcKwrmz3O2+BtjjKAvpafkmO8l7tdufThcV4q5O8DIrGKZTqPwJNl1IXNDw9bg1kWRxYtnCQ6yICmJhSFm/Y3m6xv+cXDBlHz4n/FsRC6UfTd",
          "format": "base64"
        },
        {
          "data": "MIIFYjCCBEqgAwIBAgIQd70NbNs2+RrqIQ/E8FjTDTANBgkqhkiG9w0BAQsFADBXMQswCQYDVQQGEwJCRTEZMBcGA1UEChMQR2xvYmFsU2lnbiBudi1zYTEQMA4GA1UECxMHUm9vdCBDQTEbMBkGA1UEAxMSR2xvYmFsU2lnbiBSb290IENBMB4XDTIwMDYxOTAwMDA0MloXDTI4MDEyODAwMDA0MlowRzELMAkGA1UEBhMCVVMxIjAgBgNVBAoTGUdvb2dsZSBUcnVzdCBTZXJ2aWNlcyBMTEMxFDASBgNVBAMTC0dUUyBSb290IFIxMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAthECix7joXebO9y/lD63ladAPKH9gvl9MgaCcfb2jH/76Nu8ai6Xl6OMS/kr9rH5zoQdsfnFl97vufKj6bwSiV6nqlKr+CMny6SxnGPb15l+8Ape62im9MZaRw1NEDPjTrETo8gYbEvs/AmQ351kKSUjB6G00j0uYODP0gmHu81I8E3CwnqIiru6z1kZ1q+PsAewnjHxgsHA3y6mbWwZDrXYfiYaRQM9sHmklCitD38m5agI/pboPGiUU+6DOogrFZYJsuB6jC511pzrp1Zkj5ZPaK49l8KEj8C8QMALXL32h7M1bKwYUH+E4EzNktMg6TO8UpmvMrUpsyUqtEj5cuHKZPfmghCN6J3Cioj6OGaK/GP5Afl4/Xtcd/p2h/rs37EOeZVXtL0m79YB0esWCruOC7XFxYpVq9Os6pFLKcwZpDIlTirxZUTQAs6qzkm06p98g7BAe+dDq6dso499iYH6TKX/1Y7DzkvgtdizjkXPdsDtQCv9Uw+wp9U7DbGKogPeMa3Md+pvez7W35EiEua++tgy/BBjFFFy3l3WFpO9KWgz7zpm7AeKJt8T11dleCfeXkkUAKIAf5qoIbapsZWwpbkNFhHax2xIPEDgfg1azVY80ZcFuctL7TlLnMQ/0lUTbiSw1nH69MG6zO0b9f6BQdgAmD06yK56mDcYBZUCAwEAAaOCATgwggE0MA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTkrysmcRorSCeFL1JmLO/wiRNxPjAfBgNVHSMEGDAWgBRge2YaRQ2XyolQL30EzTSo//z9SzBgBggrBgEFBQcBAQRUMFIwJQYIKwYBBQUHMAGGGWh0dHA6Ly9vY3NwLnBraS5nb29nL2dzcjEwKQYIKwYBBQUHMAKGHWh0dHA6Ly9wa2kuZ29vZy9nc3IxL2dzcjEuY3J0MDIGA1UdHwQrMCkwJ6AloCOGIWh0dHA6Ly9jcmwucGtpLmdvb2cvZ3NyMS9nc3IxLmNybDA7BgNVHSAENDAyMAgGBmeBDAECATAIBgZngQwBAgIwDQYLKwYBBAHWeQIFAwIwDQYLKwYBBAHWeQIFAwMwDQYJKoZIhvcNAQELBQADggEBADSkHrEoo9C0dhemMXoh6dFSPsjbdBZBiLg9NR3t5P+T4Vxfq7vqfM/b5A3Ri1fyJm9bvhdGaJQ3b2t6yMAYN/olUazsaL+yyEn9WprKASOshIArAoyZl+tJaox118fessmXn1hIVw41oeQa1v1vg4Fv74zPl6/AhSrw9U5pCZEt4Wi4wStz6dTZ/CLANx8LZh1J7QJVj2fhMtfTJr9w4z30Z209fOU0iOMy+qduBmpvvYuR7hZL6Dupszfnw0Skfths18dG9ZKb59UhvmaSGZRVbNQpsg3BZlvid0lIKO2d1xozclOzgjXPYovJJIultzkMu34qQb9Sz/yilrbCgj8=",
          "format": "base64"
        }
      ],
      "t": 0.086816667,
      "address": "8.8.4.4:443",
      "server_name": "dns.google",
      "alpn": [
        "h2",
        "http/1.1"
      ],
      "no_tls_verify": false,
      "oddity": "",
      "proto": "tcp",
      "started": 0.043971083
    }
  ],

  // Finally here we see information about the round trip, which
  // is formatted according to https://github.com/ooni/spec/blob/master/data-formats/df-001-httpt.md:
  "requests": [
    {

      // This field indicates whether there was an error during
      // the HTTP round trip:
      "failure": null,

      // This field contains the request method, URL, and HTTP headers
      "request": {
        "method": "GET",
        "url": "https://dns.google/",
        "headers": {
          "accept": "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
          "accept-language": "en-US;q=0.8,en;q=0.5",
          "user-agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.81 Safari/537.36"
        }
      },

      // This field contains the response status code, body, and headers.
      "response": {
        "code": 200,
        "headers": {
          "accept-ranges": "none",
          "alt-svc": "h3=\":443\"; ma=2592000,h3-29=\":443\"; ma=2592000,h3-Q050=\":443\"; ma=2592000,h3-Q046=\":443\"; ma=2592000,h3-Q043=\":443\"; ma=2592000,quic=\":443\"; ma=2592000; v=\"46,43\"",
          "cache-control": "private",
          "content-security-policy": "object-src 'none';base-uri 'self';script-src 'nonce-y/OGliLR2gbEfTG2i4MAaw==' 'strict-dynamic' 'report-sample' 'unsafe-eval' 'unsafe-inline' https: http:;report-uri https://csp.withgoogle.com/csp/honest_dns/1_0;frame-ancestors 'none'",
          "content-type": "text/html; charset=UTF-8",
          "date": "Fri, 05 Nov 2021 08:59:37 GMT",
          "server": "scaffolding on HTTPServer2",
          "strict-transport-security": "max-age=31536000; includeSubDomains; preload",
          "vary": "Accept-Encoding",
          "x-content-type-options": "nosniff",
          "x-frame-options": "SAMEORIGIN",
          "x-xss-protection": "0"
        },

        // The body in particular is a snapshot of the response
        // body: we don't want to read and submit to the OONI
        // collector large bodies.
        "body": {
          "data": "PCFET0NUWVBFIGh0bWw+CjxodG1sIGxhbmc9ImVuIj4gPGhlYWQ+IDx0aXRsZT5Hb29nbGUgUHVibGljIEROUzwvdGl0bGU+ICA8bWV0YSBjaGFyc2V0PSJVVEYtOCI+IDxsaW5rIGhyZWY9Ii9zdGF0aWMvOTNkZDU5NTQvZmF2aWNvbi5wbmciIHJlbD0ic2hvcnRjdXQgaWNvbiIgdHlwZT0iaW1hZ2UvcG5nIj4gPGxpbmsgaHJlZj0iL3N0YXRpYy84MzZhZWJjNi9tYXR0ZXIubWluLmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPGxpbmsgaHJlZj0iL3N0YXRpYy9iODUzNmMzNy9zaGFyZWQuY3NzIiByZWw9InN0eWxlc2hlZXQiPiA8bWV0YSBuYW1lPSJ2aWV3cG9ydCIgY29udGVudD0id2lkdGg9ZGV2aWNlLXdpZHRoLCBpbml0aWFsLXNjYWxlPTEiPiAgPGxpbmsgaHJlZj0iL3N0YXRpYy9kMDVjZDZiYS9yb290LmNzcyIgcmVsPSJzdHlsZXNoZWV0Ij4gPC9oZWFkPiA8Ym9keT4gPHNwYW4gY2xhc3M9ImZpbGxlciB0b3AiPjwvc3Bhbj4gICA8ZGl2IGNsYXNzPSJsb2dvIiB0aXRsZT0iR29vZ2xlIFB1YmxpYyBETlMiPiA8ZGl2IGNsYXNzPSJsb2dvLXRleHQiPjxzcGFuPlB1YmxpYyBETlM8L3NwYW4+PC9kaXY+IDwvZGl2PiAgPGZvcm0gYWN0aW9uPSIvcXVlcnkiIG1ldGhvZD0iR0VUIj4gIDxkaXYgY2xhc3M9InJvdyI+IDxsYWJlbCBjbGFzcz0ibWF0dGVyLXRleHRmaWVsZC1vdXRsaW5lZCI+IDxpbnB1dCB0eXBlPSJ0ZXh0IiBuYW1lPSJuYW1lIiBwbGFjZWhvbGRlcj0iJm5ic3A7Ij4gPHNwYW4+RE5TIE5hbWU8L3NwYW4+IDxwIGNsYXNzPSJoZWxwIj4gRW50ZXIgYSBkb21haW4gKGxpa2UgZXhhbXBsZS5jb20pIG9yIElQIGFkZHJlc3MgKGxpa2UgOC44LjguOCBvciAyMDAxOjQ4NjA6NDg2MDo6ODg0NCkgaGVyZS4gPC9wPiA8L2xhYmVsPiA8YnV0dG9uIGNsYXNzPSJtYXR0ZXItYnV0dG9uLWNvbnRhaW5lZCBtYXR0ZXItcHJpbWFyeSIgdHlwZT0ic3VibWl0Ij5SZXNvbHZlPC9idXR0b24+IDwvZGl2PiA8L2Zvcm0+ICA8c3BhbiBjbGFzcz0iZmlsbGVyIGJvdHRvbSI+PC9zcGFuPiA8Zm9vdGVyIGNsYXNzPSJyb3ciPiA8YSBocmVmPSJodHRwczovL2RldmVsb3BlcnMuZ29vZ2xlLmNvbS9zcGVlZC9wdWJsaWMtZG5zIj5IZWxwPC9hPiA8YSBocmVmPSIvY2FjaGUiPkNhY2hlIEZsdXNoPC9hPiA8c3BhbiBjbGFzcz0iZmlsbGVyIj48L3NwYW4+IDxhIGhyZWY9Imh0dHBzOi8vZGV2ZWxvcGVycy5nb29nbGUuY29tL3NwZWVkL3B1YmxpYy1kbnMvZG9jcy91c2luZyI+IEdldCBTdGFydGVkIHdpdGggR29vZ2xlIFB1YmxpYyBETlMgPC9hPiA8L2Zvb3Rlcj4gICA8c2NyaXB0IG5vbmNlPSJ5L09HbGlMUjJnYkVmVEcyaTRNQWF3PT0iPmRvY3VtZW50LmZvcm1zWzBdLm5hbWUuZm9jdXMoKTs8L3NjcmlwdD4gPC9ib2R5PiA8L2h0bWw+",
          "format": "base64"
        },

        // This field tells us whether the size of the read
        // snapshot was smaller than the snapshot size. If
        // not, then the body has been truncated.
        "body_is_truncated": false,

        // These extra fields are not part of the spec and
        // hence we prefix them with `x_`. They tell us
        // the length of the body and whether the content
        // of the body is valid UTF8.
        "x_body_length": 1383,
        "x_body_is_utf8": true
      },

      // The t field is the moment where we finished the
      // round trip and saved the event. The started field
      // is instead when we started the round trip.
      //
      // You may notice that the start of the round trip
      // if after the `t` of the handshake. This tells us
      // that the code first connects, then handshakes, and
      // finally creates HTTP code for performing the
      // round trip.
      "t": 0.471535167,
      "started": 0.087176458,

      // As usual we also compute an oddity value related
      // in this case to the HTTP round trip.
      "oddity": ""
    }
  ]
}
```

Here are some suggestions for follow up measurements:

1. provoke a connect error by using:

```
go run -race ./internal/tutorial/measurex/chapter06 -address 127.0.0.1:1 | jq
```

2. provoke a TLS handshake error by using:

```
go run -race ./internal/tutorial/measurex/chapter06 -sni example.com | jq
```

3. provoke an HTTP round trip error by using:

```
go run -race ./internal/tutorial/measurex/chapter06 -address 8.8.8.8:853 | jq
```

4. modify the code to fetch an HTTP endpoint instead (hint: you
need to change the HTTPEndpoint's URL scheme);

5. modify the code to use QUIC and HTTP/3 instead (hint: you need to
change the HTTPEndpoint's network and... is this enough?).

## Conclusion

We have seen how to measure the flow of fetching a
specific webpage from an HTTPEndpoint.

