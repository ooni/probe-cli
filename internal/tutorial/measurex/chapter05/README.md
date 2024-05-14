
# Chapter V: QUIC handshaking

This chapter describes measuring QUIC handshakes. Conceptually,
and code wise, this is very similar to the previous chapter.
The API call, in fact, has exactly the same structure, though
under the hood QUIC is different because there are no
separate connection establishment and handshake primitives.
For this reason, we will not see a connect event, but we
will only see a "QUIC handshake event".

Having said that, let us now move on and see the code of
the simple program that uses this functionality.

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

	"github.com/ooni/probe-cli/v3/internal/legacy/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	sni := flag.String("sni", "dns.google", "value for SNI extension")
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
```

### Handshaking with QUIC

The API signature is indeed the same as the previous chapter,
except that here we call the `QUICHandshake` function.

```Go
	m := mx.QUICHandshake(ctx, *address, &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		ServerName: *sni,
		NextProtos: []string{"h3"},
		RootCAs:    nil, // use netxlite's default
	})
```

The same remarks mentioned in the previous chapter regarding
the arguments for the TLS config also apply here. We need
to specify the SNI (`ServerName`), the ALPN (`NextProtos`),
and the CA pool we want to use. Here, again, we're using
the CA pool from cURL that we bundle with OONI Probe.

As we did in the previous chapters, here's the usual three
lines of code for printing the resulting measurement.

```
	data, err := json.Marshal(measurex.NewArchivalEndpointMeasurement(m))
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## Running the example program

As before, let us start off with a vanilla run:

```bash
go run -race ./internal/tutorial/measurex/chapter05 | jq
```

Produces this JSON:

```JavaScript
{
  // In chapter02 these two fields were similar but
  // the network was "tcp" as opposed to "udp"
  "network": "udp",
  "address": "8.8.4.4:443",

  // This block contains I/O operations. Note that
  // the protocol is "udp" and that the syscalls
  // are "read_from" and "write_to" because QUIC does
  // not bind/connect sockets. (The real syscalls
  // are actually `recvfrom` and `sendto` but here
  // we follow the Go convention of using read/write
  // more frequently than send/recv.)
  "network_events": [
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.027184208,
      "started": 0.027127208,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.053116458,
      "started": 0.025626583,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.054538792,
      "started": 0.054517542,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.069144958,
      "started": 0.053194208,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.069183458,
      "started": 0.069173292,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1252,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.06920225,
      "started": 0.069197875,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 1216,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.069210958,
      "started": 0.069206875,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 65,
      "operation": "read_from",
      "proto": "udp",
      "t": 0.069220667,
      "started": 0.069217375,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 44,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.069433417,
      "started": 0.069417625,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 44,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.069677625,
      "started": 0.069647458,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 83,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.073461917,
      "started": 0.073432875,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 33,
      "operation": "write_to",
      "proto": "udp",
      "t": 0.073559417,
      "started": 0.073542542,
      "oddity": ""
    }
  ],

  // This section describes the QUIC handshake and it has
  // basically the same fields as the TLS handshake.
  "quic_handshakes": [
    {
      "cipher_suite": "TLS_CHACHA20_POLY1305_SHA256",
      "failure": null,
      "negotiated_proto": "h3",
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
      "t": 0.073469208,
      "address": "8.8.4.4:443",
      "server_name": "dns.google",
      "alpn": [
        "h3"
      ],
      "no_tls_verify": false,
      "oddity": "",
      "proto": "udp",
      "started": 0.025061583
    }
  ]
}
```

Here are some suggestions for other experiments to run:

1. obtain a timeout by connecting to a port that is not
actually listening for QUIC;

2. obtain a certificate validation error by forcing
a different SNI;

3. use a different ALPN (by changing the code), and see
how the error and the oddity are handled. Can we do
anything about this by changing `./internal/netxlite/errorx`
to better support this specific error condition?

## Conclusion

We have seen how to perform QUIC handshake and
collect measurements.

