
# Chapter IV: TLS handshaking

This chapter describes measuring TLS handshakes.

Without further ado, let's describe our example `main.go` program
and let's use it to better understand how to measure that.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter04/main.go`.)

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
	sni := flag.String("sni", "dns.google", "domain to resolver")
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
```

### Connecting and handshaking.

We call the `ConnectAndHandshake` method. The arguments
are the context, the address, and a TLS config.

Under the hood, the code will call the TCP connect functionality
we have seen in chapter02, using the address argument. Then, if
successful, it will TLS handshake using the given TLS config.

```Go
	m := mx.TLSConnectAndHandshake(ctx, *address, &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    nil, // use netxlite's default
	})
```

Regarding the TLS config, in particular,
the three fields above are the files you should always set
in a TLS config when doing handshakes manually. The `ServerName`
field forces the SNI, the NextProtos field forces the ALPN,
and the `RootCAs` field is overridden so that we use the
CA bundle that we bundle with OONI. (This CA bundle is the
same you can find at https://curl.haxx.se/ca/.)

As usual, the method to perform a measurement returns
the measurement itself, which we print below.

```
	data, err := json.Marshal(measurex.NewArchivalEndpointMeasurement(m))
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```
## Running the example program

As before, let us start off with a vanilla run:

```bash
go run -race ./internal/tutorial/measurex/chapter04 | jq
```

Let us comment the JSON in detail:

```JavaScript
{
  "network": "tcp",
  "address": "8.8.4.4:443",

  // These are the I/O events during the handshake
  "network_events": [
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 280,
      "operation": "write",
      "proto": "tcp",
      "t": 0.048268333,
      "started": 0.048246666,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 517,
      "operation": "read",
      "proto": "tcp",
      "t": 0.086214708,
      "started": 0.048287791,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 4303,
      "operation": "read",
      "proto": "tcp",
      "t": 0.087951708,
      "started": 0.08792725,
      "oddity": ""
    },
    {
      "address": "8.8.4.4:443",
      "failure": null,
      "num_bytes": 64,
      "operation": "write",
      "proto": "tcp",
      "t": 0.090097833,
      "started": 0.090083875,
      "oddity": ""
    }
  ],

  // This block is generated when connecting to a TCP
  // socket, as we've already seen in chapter02
  "tcp_connect": [
    {
      "ip": "8.8.4.4",
      "port": 443,
      "t": 0.046022583,
      "status": {
        "blocked": false,
        "failure": null,
        "success": true
      },
      "started": 0.024424916,
      "oddity": ""
    }
  ],

  // This block contains information about the handshake.
  //
  // See https://github.com/ooni/spec/blob/master/data-formats/df-006-tlshandshake.md
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
      "t": 0.090150666,
      "address": "8.8.4.4:443",
      "server_name": "dns.google",
      "alpn": [
        "h2",
        "http/1.1"
      ],
      "no_tls_verify": false,
      "oddity": "",
      "proto": "tcp",
      "started": 0.04635975
    }
  ]
}
```

### Suggested follow-up experiments

Try to run experiments in the following scenarios, and
check the output JSON to familiarize yourself with what
changes in different error conditions.

1. measurement that causes timeout

2. measurement with wrong SNI

3. measurement with self-signed certificate

4. measurement with expired certificate

5. measurement with connection reset during handshake

6. measurement with timeout during handshake

Here are the commands I used for each proposed exercise:

1. go run -race ./internal/tutorial/measurex/chapter04 -address 8.8.4.4:1 | jq

2. go run -race ./internal/tutorial/measurex/chapter04 -sni example.org | jq

3. go run -race ./internal/tutorial/measurex/chapter04 -address 104.154.89.105:443 -sni self-signed.badssl.com | jq

4. go run -race ./internal/tutorial/measurex/chapter04 -address 104.154.89.105:443 -sni expire.badssl.com | jq

To emulate the last two scenarios, if you're on Linux, a
possibility is building Jafar with this command:

```
go build -v ./internal/cmd/tinyjafar
```

Then, for example, to provoke a connection reset you
can run in a terminal:

```
sudo ./tinyjafar -iptables-reset-keyword dns.google
```

and you can run this tutorial with `dns.google` as
the SNI in another terminal.

Likewise, you can obtain a timeout using the
`-iptables-drop-keyword` flag instead.

(Jafar runs forever and censors. You need to use
`^C` to terminate it from running.)

## Conclusion

We have seen how to measure TLS handshakes. We have seen how
this flow produces a different output on different error conditions.

