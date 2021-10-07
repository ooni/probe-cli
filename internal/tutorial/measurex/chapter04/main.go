// -=-=- StartHere -=-=-
//
// # Chapter IV: TLS handshaking
//
// This chapter describes measuring TLS handshakes.
//
// Without further ado, let's describe our example `main.go` program
// and let's use it to better understand how to measure that.
//
// (This file is auto-generated. Do not edit it directly! To apply
// changes you need to modify `./internal/tutorial/measurex/chapter04/main.go`.)
//
// ## main.go
//
// The initial part of the program is pretty much the same as the one
// used in previous chapters, so I will not add further comments.
//
// ```Go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	// ```
	//
	// ### Connecting and handshaking.
	//
	// We call the `ConnectAndHandshake` method. The arguments
	// are the context, the address, and a TLS config.
	//
	// Under the hood, the code will call the TCP connect functionality
	// we have seen in chapter02, using the address argument. Then, if
	// successful, it will TLS handshake using the given TLS config.
	//
	// ```Go
	m := mx.TLSConnectAndHandshake(ctx, *address, &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	// ```
	//
	// Regarding the TLS config, in particular,
	// the three fields above are the files you should always set
	// in a TLS config when doing handshakes manually. The `ServerName`
	// field forces the SNI, the NextProtos field forces the ALPN,
	// and the `RootCAs` field is overridden so that we use the
	// CA bundle that we bundle with OONI. (This CA bundle is the
	// same you can find at https://curl.haxx.se/ca/.)
	//
	// As usual, the method to perform a measurement returns
	// the measurement itself, which we print below.
	//
	// ```
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

// ```
// ## Running the example program
//
// As before, let us start off with a vanilla run:
//
// ```bash
// go run -race ./internal/tutorial/measurex/chapter04
// ```
//
// Let us comment the JSON in detail:
//
// ```JavaScript
// {
//   "network": "tcp",
//   "address": "8.8.4.4:443",
//
//   // This block is generated when connecting to a TCP
//   // socket, as we've already seen in chapter02
//   "connect": [
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "operation": "connect",
//       "proto": "tcp",
//       "t": 0.046959084,
//       "started": 0.022998875,
//       "oddity": ""
//     }
//   ],
//
//   // These are the I/O events during the handshake
//   "read_write": [
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "num_bytes": 280,
//       "operation": "write",
//       "proto": "tcp",
//       "t": 0.048752875,
//       "started": 0.04874125,
//       "oddity": ""
//     },
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "num_bytes": 517,
//       "operation": "read",
//       "proto": "tcp",
//       "t": 0.087221334,
//       "started": 0.048760417,
//       "oddity": ""
//     },
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "num_bytes": 4301,
//       "operation": "read",
//       "proto": "tcp",
//       "t": 0.088843584,
//       "started": 0.088830959,
//       "oddity": ""
//     },
//     {
//       "address": "8.8.4.4:443",
//       "failure": null,
//       "num_bytes": 64,
//       "operation": "write",
//       "proto": "tcp",
//       "t": 0.092078042,
//       "started": 0.092064042,
//       "oddity": ""
//     }
//   ],
//
//   // This block contains information about the handshake
//   "tls_handshake": [
//     {
//       "cipher_suite": "TLS_AES_128_GCM_SHA256",
//       "failure": null,
//       "negotiated_proto": "h2",
//       "tls_version": "TLSv1.3",
//       "peer_certificates": [
//         {
//           "data": "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",
//           "format": "base64"
//         },
//         {
//           "data": "MIIFljCCA36gAwIBAgINAgO8U1lrNMcY9QFQZjANBgkqhkiG9w0BAQsFADBHMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzEUMBIGA1UEAxMLR1RTIFJvb3QgUjEwHhcNMjAwODEzMDAwMDQyWhcNMjcwOTMwMDAwMDQyWjBGMQswCQYDVQQGEwJVUzEiMCAGA1UEChMZR29vZ2xlIFRydXN0IFNlcnZpY2VzIExMQzETMBEGA1UEAxMKR1RTIENBIDFDMzCCASIwDQYJKoZIhvcNAQEBBQADggEPADCCAQoCggEBAPWI3+dijB43+DdCkH9sh9D7ZYIl/ejLa6T/belaI+KZ9hzpkgOZE3wJCor6QtZeViSqejOEH9Hpabu5dOxXTGZok3c3VVP+ORBNtzS7XyV3NzsXlOo85Z3VvMO0Q+sup0fvsEQRY9i0QYXdQTBIkxu/t/bgRQIh4JZCF8/ZK2VWNAcmBA2o/X3KLu/qSHw3TT8An4Pf73WELnlXXPxXbhqW//yMmqaZviXZf5YsBvcRKgKAgOtjGDxQSYflispfGStZloEAoPtR28p3CwvJlk/vcEnHXG0g/Zm0tOLKLnf9LdwLtmsTDIwZKxeWmLnwi/agJ7u2441Rj72ux5uxiZ0CAwEAAaOCAYAwggF8MA4GA1UdDwEB/wQEAwIBhjAdBgNVHSUEFjAUBggrBgEFBQcDAQYIKwYBBQUHAwIwEgYDVR0TAQH/BAgwBgEB/wIBADAdBgNVHQ4EFgQUinR/r4XN7pXNPZzQ4kYU83E1HScwHwYDVR0jBBgwFoAU5K8rJnEaK0gnhS9SZizv8IkTcT4waAYIKwYBBQUHAQEEXDBaMCYGCCsGAQUFBzABhhpodHRwOi8vb2NzcC5wa2kuZ29vZy9ndHNyMTAwBggrBgEFBQcwAoYkaHR0cDovL3BraS5nb29nL3JlcG8vY2VydHMvZ3RzcjEuZGVyMDQGA1UdHwQtMCswKaAnoCWGI2h0dHA6Ly9jcmwucGtpLmdvb2cvZ3RzcjEvZ3RzcjEuY3JsMFcGA1UdIARQME4wOAYKKwYBBAHWeQIFAzAqMCgGCCsGAQUFBwIBFhxodHRwczovL3BraS5nb29nL3JlcG9zaXRvcnkvMAgGBmeBDAECATAIBgZngQwBAgIwDQYJKoZIhvcNAQELBQADggIBAIl9rCBcDDy+mqhXlRu0rvqrpXJxtDaV/d9AEQNMwkYUuxQkq/BQcSLbrcRuf8/xam/IgxvYzolfh2yHuKkMo5uhYpSTld9brmYZCwKWnvy15xBpPnrLRklfRuFBsdeYTWU0AIAaP0+fbH9JAIFTQaSSIYKCGvGjRFsqUBITTcFTNvNCCK9U+o53UxtkOCcXCb1YyRt8OS1b887U7ZfbFAO/CVMkH8IMBHmYJvJh8VNS/UKMG2YrPxWhu//2m+OBmgEGcYk1KCTd4b3rGS3hSMs9WYNRtHTGnXzGsYZbr8w0xNPM1IERlQCh9BIiAfq0g3GvjLeMcySsN1PCAJA/Ef5c7TaUEDu9Ka7ixzpiO2xj2YC/WXGsYye5TBeg2vZzFb8q3o/zpWwygTMD0IZRcZk0upONXbVRWPeyk+gB9lm+cZv9TSjOz23HFtz30dZGm6fKa+l3D/2gthsjgx0QGtkJAITgRNOidSOzNIb2ILCkXhAd4FJGAJ2xDx8hcFH1mt0G/FX0Kw4zd8NLQsLxdxP8c4CU6x+7Nz/OAipmsHMdMqUybDKwjuDEI/9bfU1lcKwrmz3O2+BtjjKAvpafkmO8l7tdufThcV4q5O8DIrGKZTqPwJNl1IXNDw9bg1kWRxYtnCQ6yICmJhSFm/Y3m6xv+cXDBlHz4n/FsRC6UfTd",
//           "format": "base64"
//         },
//         {
//           "data": "MIIFYjCCBEqgAwIBAgIQd70NbNs2+RrqIQ/E8FjTDTANBgkqhkiG9w0BAQsFADBXMQswCQYDVQQGEwJCRTEZMBcGA1UEChMQR2xvYmFsU2lnbiBudi1zYTEQMA4GA1UECxMHUm9vdCBDQTEbMBkGA1UEAxMSR2xvYmFsU2lnbiBSb290IENBMB4XDTIwMDYxOTAwMDA0MloXDTI4MDEyODAwMDA0MlowRzELMAkGA1UEBhMCVVMxIjAgBgNVBAoTGUdvb2dsZSBUcnVzdCBTZXJ2aWNlcyBMTEMxFDASBgNVBAMTC0dUUyBSb290IFIxMIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAthECix7joXebO9y/lD63ladAPKH9gvl9MgaCcfb2jH/76Nu8ai6Xl6OMS/kr9rH5zoQdsfnFl97vufKj6bwSiV6nqlKr+CMny6SxnGPb15l+8Ape62im9MZaRw1NEDPjTrETo8gYbEvs/AmQ351kKSUjB6G00j0uYODP0gmHu81I8E3CwnqIiru6z1kZ1q+PsAewnjHxgsHA3y6mbWwZDrXYfiYaRQM9sHmklCitD38m5agI/pboPGiUU+6DOogrFZYJsuB6jC511pzrp1Zkj5ZPaK49l8KEj8C8QMALXL32h7M1bKwYUH+E4EzNktMg6TO8UpmvMrUpsyUqtEj5cuHKZPfmghCN6J3Cioj6OGaK/GP5Afl4/Xtcd/p2h/rs37EOeZVXtL0m79YB0esWCruOC7XFxYpVq9Os6pFLKcwZpDIlTirxZUTQAs6qzkm06p98g7BAe+dDq6dso499iYH6TKX/1Y7DzkvgtdizjkXPdsDtQCv9Uw+wp9U7DbGKogPeMa3Md+pvez7W35EiEua++tgy/BBjFFFy3l3WFpO9KWgz7zpm7AeKJt8T11dleCfeXkkUAKIAf5qoIbapsZWwpbkNFhHax2xIPEDgfg1azVY80ZcFuctL7TlLnMQ/0lUTbiSw1nH69MG6zO0b9f6BQdgAmD06yK56mDcYBZUCAwEAAaOCATgwggE0MA4GA1UdDwEB/wQEAwIBhjAPBgNVHRMBAf8EBTADAQH/MB0GA1UdDgQWBBTkrysmcRorSCeFL1JmLO/wiRNxPjAfBgNVHSMEGDAWgBRge2YaRQ2XyolQL30EzTSo//z9SzBgBggrBgEFBQcBAQRUMFIwJQYIKwYBBQUHMAGGGWh0dHA6Ly9vY3NwLnBraS5nb29nL2dzcjEwKQYIKwYBBQUHMAKGHWh0dHA6Ly9wa2kuZ29vZy9nc3IxL2dzcjEuY3J0MDIGA1UdHwQrMCkwJ6AloCOGIWh0dHA6Ly9jcmwucGtpLmdvb2cvZ3NyMS9nc3IxLmNybDA7BgNVHSAENDAyMAgGBmeBDAECATAIBgZngQwBAgIwDQYLKwYBBAHWeQIFAwIwDQYLKwYBBAHWeQIFAwMwDQYJKoZIhvcNAQELBQADggEBADSkHrEoo9C0dhemMXoh6dFSPsjbdBZBiLg9NR3t5P+T4Vxfq7vqfM/b5A3Ri1fyJm9bvhdGaJQ3b2t6yMAYN/olUazsaL+yyEn9WprKASOshIArAoyZl+tJaox118fessmXn1hIVw41oeQa1v1vg4Fv74zPl6/AhSrw9U5pCZEt4Wi4wStz6dTZ/CLANx8LZh1J7QJVj2fhMtfTJr9w4z30Z209fOU0iOMy+qduBmpvvYuR7hZL6Dupszfnw0Skfths18dG9ZKb59UhvmaSGZRVbNQpsg3BZlvid0lIKO2d1xozclOzgjXPYovJJIultzkMu34qQb9Sz/yilrbCgj8=",
//           "format": "base64"
//         }
//       ],
//       "t": 0.092117709,
//       "address": "8.8.4.4:443",
//       "server_name": "dns.google",
//       "alpn": [
//         "h2",
//         "http/1.1"
//       ],
//       "no_tls_verify": false,
//       "oddity": "",
//       "proto": "tcp",
//       "started": 0.047288542
//     }
//   ]
// }
// ```
//
// All the data formats we're using here are, by the way,
// compatible with the data formats specified at
// https://github.com/ooni/spec/tree/master/data-formats.
//
// ### Suggested follow-up experiments
//
// Try to run experiments in the following scenarios, and
// check the output JSON to familiarize yourself with what 
// changes in different error conditions.
//
// 1. measurement that causes timeout
//
// 2. measurement with wrong SNI
//
// 3. measurement with self-signed certificate
//
// 4. measurement with expired certificate
//
// 5. measurement with connection reset during handshake
//
// 6. measurement with timeout during handshake
//
// Here are the commands I used for each proposed exercise:
//
// 1. go run -race ./internal/tutorial/measurex/chapter04 -address 8.8.4.4:1
//
// 2. go run -race ./internal/tutorial/measurex/chapter04 -sni example.org
//
// 3. go run -race ./internal/tutorial/measurex/chapter04 -address 104.154.89.105:443 -sni self-signed.badssl.com
//
// 4. go run -race ./internal/tutorial/measurex/chapter04 -address 104.154.89.105:443 -sni expire.badssl.com
//
// To emulate the last two scenarios, if you're on Linux, a
// possibility is building Jafar with this command:
//
// ```
// go build -v ./internal/cmd/jafar
// ```
//
// Then, for example, to provoke a connection reset you
// can run in a terminal:
//
// ```
// sudo ./jafar -iptables-reset-keyword dns.google
// ```
//
// and you can run this tutorial with `dns.google` as
// the SNI in another terminal.
//
// Likewise, you can obtain a timeout using the
// `-iptables-drop-keyword` flag instead.
//
// (Jafar runs forever and censors. You need to use
// `^C` to terminate it from running.)
//
// ## Conclusion
//
// We have seen how to measure TLS handshakes. We have seen how
// this flow produces a different output on different error conditions.
//
// -=-=- StopHere -=-=-
