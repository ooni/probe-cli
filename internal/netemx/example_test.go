package netemx_test

import (
	"bytes"
	"context"
	"fmt"
	"net"
	"net/http"
	"net/url"

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
		netemx.QAEnvOptionNetStack("8.8.4.4", &netemx.DNSOverUDPServerFactory{}),
		netemx.QAEnvOptionNetStack("9.9.9.9", &netemx.DNSOverUDPServerFactory{}),
		netemx.QAEnvOptionClientAddress(netemx.DefaultClientAddress),
		netemx.QAEnvOptionNetStack(
			netemx.AddressWwwExampleCom,
			&netemx.HTTPCleartextServerFactory{
				Factory: netemx.ExampleWebPageHandlerFactory(),
				Ports:   []int{80},
			},
			&netemx.HTTPSecureServerFactory{
				Factory:          netemx.ExampleWebPageHandlerFactory(),
				Ports:            []int{443},
				ServerNameMain:   "www.example.com",
				ServerNameExtras: []string{},
			},
			&netemx.HTTP3ServerFactory{
				Factory:          netemx.ExampleWebPageHandlerFactory(),
				Ports:            []int{443},
				ServerNameMain:   "www.example.com",
				ServerNameExtras: []string{},
			},
		),
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
		// TODO(https://github.com/ooni/probe/issues/2534): the NewHTTPClientWithResolver func has QUIRKS but we don't care.
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
		netemx.QAEnvOptionNetStack(e1WhatsappNet, netemx.NewTCPEchoServerFactory(log.Log, 5222)),
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

// This example shows how the [InternetScenario] defines DNS-over-HTTPS and DNS-over-UDP servers.
func Example_dohWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		for _, domain := range []string{"mozilla.cloudflare-dns.com", "dns.google", "dns.quad9.net"} {
			// DNS-over-UDP
			{
				dialer := netxlite.NewDialerWithResolver(log.Log, netxlite.NewStdlibResolver(log.Log))
				reso := netxlite.NewParallelUDPResolver(log.Log, dialer, net.JoinHostPort(domain, "53"))
				defer reso.CloseIdleConnections()

				addrs, err := reso.LookupHost(context.Background(), "www.example.com")
				if err != nil {
					log.Fatalf("reso.LookupHost failed: %s", err.Error())
				}

				fmt.Printf("%+v\n", addrs)
			}

			// DNS-over-HTTPS
			{
				URL := &url.URL{Scheme: "https", Host: domain, Path: "/dns-query"}
				reso := netxlite.NewParallelDNSOverHTTPSResolver(log.Log, URL.String())
				defer reso.CloseIdleConnections()

				addrs, err := reso.LookupHost(context.Background(), "www.example.com")
				if err != nil {
					log.Fatalf("reso.LookupHost failed: %s", err.Error())
				}

				fmt.Printf("%+v\n", addrs)
			}
		}
	})

	// Output:
	// [93.184.216.34]
	// [93.184.216.34]
	// [93.184.216.34]
	// [93.184.216.34]
	// [93.184.216.34]
	// [93.184.216.34]
}

// This example shows how the [InternetScenario] defines DNS-over-UDP servers.
func Example_dnsOverUDPWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		resolvers := []string{
			net.JoinHostPort(netemx.ISPResolverAddress, "53"),
			net.JoinHostPort(netemx.RootResolverAddress, "53"),
			net.JoinHostPort(netemx.AddressDNSGoogle8844, "53"),
			net.JoinHostPort(netemx.AddressDNSGoogle8888, "53"),
			net.JoinHostPort(netemx.AddressDNSQuad9Net, "53"),
			net.JoinHostPort(netemx.AddressMozillaCloudflareDNSCom, "53"),
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
	// [93.184.216.34]
	// [93.184.216.34]
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
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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

		// simplify comparison by stripping all the leading whitespaces
		simplifyBody := func(body []byte) (output []byte) {
			lines := bytes.Split(body, []byte("\n"))
			for _, line := range lines {
				line = bytes.TrimSpace(line)
				line = append(line, '\n')
				output = append(output, line...)
			}
			return output
		}

		fmt.Printf("%+v\n", string(simplifyBody(body)))
	})

	// Output:
	// <!doctype html>
	// <html>
	// <head>
	// <title>Default Web Page</title>
	// </head>
	// <body>
	// <div>
	// <h1>Default Web Page</h1>
	//
	// <p>This is the default web page of the default domain.</p>
	//
	// <p>We detect webpage blocking by checking for the status code first. If the status
	// code is different, we consider the measurement http-diff. On the contrary when
	// the status code matches, we say it's all good if one of the following check succeeds:</p>
	//
	// <p><ol>
	// <li>the body length does not match (we say they match is the smaller of the two
	// webpages is 70% or more of the size of the larger webpage);</li>
	//
	// <li>the uncommon headers match;</li>
	//
	// <li>the webpage title contains mostly the same words.</li>
	// </ol></p>
	//
	// <p>If the three above checks fail, then we also say that there is http-diff. Because
	// we need QA checks to work as intended, the size of THIS webpage you are reading
	// has been increased, by adding this description, such that the body length check fails. The
	// original webpage size was too close to the blockpage in size, and therefore we did see
	// that there was no http-diff, as it ought to be.</p>
	//
	// <p>To make sure we're not going to have this issue in the future, there is now a runtime
	// check that causes our code to crash if this web page size is too similar to the one of
	// the default blockpage. We chose to add this text for additional clarity.</p>
	//
	// <p>Also, note that the blockpage MUST be very small, because in some cases we need
	// to spoof it into a single TCP segment using ooni/netem's DPI.</p>
	// </div>
	// </body>
	// </html>
}

// This example shows how the [InternetScenario] defines an OONI-API-like service.
func Example_ooniAPIWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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
	// {"tcp_connect":{"93.184.216.34:443":{"status":true,"failure":null}},"tls_handshake":{"93.184.216.34:443":{"server_name":"www.example.com","status":true,"failure":null}},"quic_handshake":{},"http_request":{"body_length":1533,"discovered_h3_endpoint":"www.example.com:443","failure":null,"title":"Default Web Page","headers":{"Alt-Svc":"h3=\":443\"","Content-Length":"1533","Content-Type":"text/html; charset=utf-8","Date":"Thu, 24 Aug 2023 14:35:29 GMT"},"status_code":200},"http3_request":null,"dns":{"failure":null,"addrs":["93.184.216.34"]},"ip_info":{"93.184.216.34":{"asn":15133,"flags":11}}}
}

// This example shows how the [InternetScenario] defines a GeoIP service like Ubuntu's one.
func Example_ubuntuGeoIPWithInternetScenario() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
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

// This example shows how the [InternetScenario] defines a public blockpage server.
func Example_examplePublicBlockpage() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPClientStdlib has QUIRKS but they're not needed here
		client := netxlite.NewHTTPClientStdlib(log.Log)

		req, err := http.NewRequest("GET", "http://"+netemx.AddressPublicBlockpage+"/", nil)
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
	// 	<title>Access Denied</title>
	// </head>
	// <body>
	// <div>
	// 	<h1>Access Denied</h1>
	// 	<p>This request cannot be served in your jurisdiction.</p>
	// </div>
	// </body>
	// </html>
}

// This example shows how the [InternetScenario] includes an URL shortener.
func Example_exampleURLShortener() {
	env := netemx.MustNewScenario(netemx.InternetScenario)
	defer env.Close()

	env.Do(func() {
		// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPTransportStdlib has QUIRKS but we
		// don't actually care about those QUIRKS in this context
		client := netxlite.NewHTTPTransportStdlib(log.Log)

		req, err := http.NewRequest("GET", "https://bit.ly/21645", nil)
		if err != nil {
			log.Fatalf("http.NewRequest failed: %s", err.Error())
		}

		resp, err := client.RoundTrip(req)
		if err != nil {
			log.Fatalf("client.Do failed: %s", err.Error())
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusPermanentRedirect {
			log.Fatalf("got unexpected status code: %d", resp.StatusCode)
		}

		fmt.Printf("%+v\n", resp.Header.Get("Location"))
	})

	// Output:
	// https://www.example.com/
}
