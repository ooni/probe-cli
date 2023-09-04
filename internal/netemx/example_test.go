package netemx_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// exampleNewEnvironment creates a QA environment setting all possible options. We're going
// to use this QA environment in all the examples for this package.
func exampleNewEnvironment() *netemx.QAEnv {
	return netemx.MustNewQAEnv(
		netemx.QAEnvOptionDNSOverUDPResolvers("8.8.4.4", "9.9.9.9"),
		netemx.QAEnvOptionClientAddress(netemx.DefaultClientAddress),
		netemx.QAEnvOptionISPResolverAddress(netemx.DefaultISPResolverAddress),
		netemx.QAEnvOptionHTTPServer(
			netemx.AddressWwwExampleCom, netemx.ExampleWebPageHandlerFactory()),
		netemx.QAEnvOptionLogger(log.Log),
	)
}

// exampleAddRecordToAllResolvers shows how to add a DNS record to all the resolvers (i.e., including
// both the custom created resolvers and the ISP specific resolver).
func exampleAddRecordToAllResolvers(env *netemx.QAEnv) {
	env.AddRecordToAllResolvers(
		"example.com",
		"", // CNAME
		netemx.AddressWwwExampleCom,
	)
}

// This example shows how to configure a DPI rule for a QA environment.
func Example_dpiRule() {
	// create the QA environment
	env := exampleNewEnvironment()

	// make sure we close all the resources when we're done
	defer env.Close()

	// create common DNS configuration for clients and servers
	exampleAddRecordToAllResolvers(env)

	// install a DPI rule in the environment
	dpi := env.DPIEngine()
	dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
		Logger: model.DiscardLogger,
		SNI:    "example.com",
	})

	// collect the overall error
	var err error

	// run netxlite code inside the netemx environment
	env.Do(func() {
		// create a system resolver instance
		reso := netxlite.NewStdlibResolver(model.DiscardLogger)

		// create the HTTP client
		client := netxlite.NewHTTPClientWithResolver(model.DiscardLogger, reso)

		// create the HTTP request
		req := runtimex.Try1(http.NewRequest("GET", "https://example.com", nil))

		// obtain the HTTP response or error
		_, err = client.Do(req)
	})

	// print the error that we received
	fmt.Println(err)

	// Output:
	// connection_reset
}

// This example shows how to configure different resolvers to reply differently
func Example_resolverConfig() {
	// create the QA environment
	env := exampleNewEnvironment()

	// make sure we close all the resources when we're done
	defer env.Close()

	// create a configuration for the uncensored resolvers in the network
	env.OtherResolversConfig().AddRecord(
		"example.com",
		"", // CNAME
		netemx.AddressWwwExampleCom,
	)

	// create a censored configuration for getaddrinfo
	env.ISPResolverConfig().AddRecord(
		"example.com",
		"",
		"10.10.34.35",
	)

	// collect the overall results
	var (
		googleResults []string
		quad9Results  []string
		ispResults    []string
	)

	// run netxlite code inside the netemx environment
	env.Do(func() {
		// use a system resolver instance
		{
			reso := netxlite.NewStdlibResolver(log.Log)
			ispResults = runtimex.Try1(reso.LookupHost(context.Background(), "example.com"))
		}

		// use 8.8.4.4
		{
			dialer := netxlite.NewDialerWithoutResolver(log.Log)
			reso := netxlite.NewParallelUDPResolver(log.Log, dialer, "8.8.4.4:53")
			googleResults = runtimex.Try1(reso.LookupHost(context.Background(), "example.com"))
		}

		// use 9.9.9.9
		{
			dialer := netxlite.NewDialerWithoutResolver(log.Log)
			reso := netxlite.NewParallelUDPResolver(log.Log, dialer, "9.9.9.9:53")
			quad9Results = runtimex.Try1(reso.LookupHost(context.Background(), "example.com"))
		}
	})

	// print the results that we received
	fmt.Println(googleResults, quad9Results, ispResults)

	// Output:
	// [93.184.216.34] [93.184.216.34] [10.10.34.35]
}

// This example shows how to create a TCP listener attached to an arbitrary netstack handler.
func Example_customNetStackHandler() {
	// e1WhatsappNet is e1.whatsapp.net IP address as of 2023-07-11
	const e1WhatsappNet = "3.33.252.61"

	// create the QA environment
	env := netemx.MustNewQAEnv(
		netemx.QAEnvOptionNetStack(e1WhatsappNet, netemx.TCPEchoNetStack(log.Log, 5222)),
		netemx.QAEnvOptionLogger(log.Log),
	)

	// make sure we close all the resources when we're done
	defer env.Close()

	// create common DNS configuration for clients and servers
	env.AddRecordToAllResolvers("e1.whatsapp.net", "", e1WhatsappNet)

	// run netxlite code inside the netemx environment
	env.Do(func() {
		// create a system resolver instance
		reso := netxlite.NewStdlibResolver(log.Log)

		// create a dialer
		dialer := netxlite.NewDialerWithResolver(log.Log, reso)

		// attempt to establish a TCP connection
		conn, err := dialer.DialContext(context.Background(), "tcp", "e1.whatsapp.net:5222")

		// make sure no error occurred
		if err != nil {
			log.Fatalf("dialer.DialContext failed: %s", err.Error())
		}

		// send data to the echo server
		input := []byte("0xdeadbeef")
		if _, err := conn.Write(input); err != nil {
			log.Fatalf("conn.Write failed: %s", err.Error())
		}

		// receive data from the echo server
		buffer := make([]byte, 1<<17)
		count, err := conn.Read(buffer)
		if err != nil {
			log.Fatalf("conn.Read failed: %s", err.Error())
		}
		output := buffer[:count]

		// print whether input and output are equal
		fmt.Println(bytes.Equal(input, output))

		// close the connection
		conn.Close()
	})

	// Output:
	// true
}

// This example shows how the [InternetScenario] defines DoH servers.
func Example_dohWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		reso := netxlite.NewParallelDNSOverHTTPSResolver(log.Log, "https://dns.google/dns-query")
		defer reso.CloseIdleConnections()

		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err != nil {
			log.Fatalf("reso.LookupHost failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", addrs)
	})

	// Output:
	// [93.184.216.34]
}

// This example shows how the [InternetScenario] defines DNS-over-UDP servers.
func Example_dnsOverUDPWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		resolvers := []string{
			net.JoinHostPort(netemx.DefaultISPResolverAddress, "53"),
			net.JoinHostPort(netemx.DefaultUncensoredResolverAddress, "53"),
		}

		for _, endpoint := range resolvers {
			dialer := netxlite.NewDialerWithoutResolver(log.Log)
			reso := netxlite.NewParallelUDPResolver(log.Log, dialer, endpoint)
			defer reso.CloseIdleConnections()

			addrs, err := reso.LookupHost(context.Background(), "www.example.com")
			if err != nil {
				log.Fatalf("reso.LookupHost failed: %s", err.Error())
			}

			fmt.Printf("%+v\n", addrs)
		}
	})

	// Output:
	// [93.184.216.34]
	// [93.184.216.34]
}

// This example shows how the [InternetScenario] supports calling getaddrinfo.
func Example_getaddrinfoWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		reso := netxlite.NewStdlibResolver(log.Log)
		defer reso.CloseIdleConnections()

		addrs, err := reso.LookupHost(context.Background(), "www.example.com")
		if err != nil {
			log.Fatalf("reso.LookupHost failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", addrs)
	})

	// Output:
	// [93.184.216.34]
}

// This example shows how the [InternetScenario] defines an example.com-like webserver.
func Example_exampleWebServerWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)

		req, err := http.NewRequest("GET", "https://www.example.com/", nil)
		if err != nil {
			log.Fatalf("http.NewRequest failed: %s", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("client.Do failed: %s", err.Error())
		}
		defer resp.Body.Close()
		body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
		if err != nil {
			log.Fatalf("netxlite.ReadAllContext failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", string(body))
	})

	// Output:
	// <!doctype html>
	// <html>
	// <head>
	// 	<title>Default Web Page</title>
	// </head>
	// <body>
	// <div>
	// 	<h1>Default Web Page</h1>
	// 	<p>This is the default web page of the default domain.</p>
	// </div>
	// </body>
	// </html>
}

// This example shows how the [InternetScenario] defines an OONI-API-like service.
func Example_ooniAPIWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)

		req, err := http.NewRequest("GET", "https://api.ooni.io/api/v1/test-helpers", nil)
		if err != nil {
			log.Fatalf("http.NewRequest failed: %s", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("client.Do failed: %s", err.Error())
		}
		defer resp.Body.Close()
		body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
		if err != nil {
			log.Fatalf("netxlite.ReadAllContext failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", string(body))
	})

	// Output:
	// {"web-connectivity":[{"address":"https://2.th.ooni.org","type":"https"},{"address":"https://3.th.ooni.org","type":"https"},{"address":"https://0.th.ooni.org","type":"https"},{"address":"https://1.th.ooni.org","type":"https"}]}
}

// This example shows how the [InternetScenario] defines an oohelperd instance.
func Example_oohelperdWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)
		thRequest := []byte(`{"http_request": "https://www.example.com/","http_request_headers":{},"tcp_connect":["93.184.216.34:443"]}`)

		req, err := http.NewRequest("POST", "https://0.th.ooni.org/", bytes.NewReader(thRequest))
		if err != nil {
			log.Fatalf("http.NewRequest failed: %s", err.Error())
		}

		log.SetLevel(log.DebugLevel)

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("client.Do failed: %s", err.Error())
		}
		defer resp.Body.Close()
		body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
		if err != nil {
			log.Fatalf("netxlite.ReadAllContext failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", string(body))
	})

	// Output:
	// {"tcp_connect":{"93.184.216.34:443":{"status":true,"failure":null}},"tls_handshake":{"93.184.216.34:443":{"server_name":"www.example.com","status":true,"failure":null}},"quic_handshake":{},"http_request":{"body_length":194,"discovered_h3_endpoint":"www.example.com:443","failure":null,"title":"Default Web Page","headers":{"Alt-Svc":"h3=\":443\"","Content-Length":"194","Content-Type":"text/html; charset=utf-8","Date":"Thu, 24 Aug 2023 14:35:29 GMT"},"status_code":200},"http3_request":null,"dns":{"failure":null,"addrs":["93.184.216.34"]},"ip_info":{"93.184.216.34":{"asn":15133,"flags":11}}}
}

// This example shows how the [InternetScenario] defines a GeoIP service like Ubuntu's one.
func Example_ubuntuGeoIPWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		client := netxlite.NewHTTPClientStdlib(log.Log)

		req, err := http.NewRequest("GET", "https://geoip.ubuntu.com/lookup", nil)
		if err != nil {
			log.Fatalf("http.NewRequest failed: %s", err.Error())
		}

		resp, err := client.Do(req)
		if err != nil {
			log.Fatalf("client.Do failed: %s", err.Error())
		}
		defer resp.Body.Close()
		body, err := netxlite.ReadAllContext(req.Context(), resp.Body)
		if err != nil {
			log.Fatalf("netxlite.ReadAllContext failed: %s", err.Error())
		}

		fmt.Printf("%+v\n", string(body))
	})

	// Output:
	// <?xml version="1.0" encoding="UTF-8"?><Response><Ip>130.192.91.211</Ip></Response>
}
