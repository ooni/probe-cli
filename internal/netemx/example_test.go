package netemx_test

import (
	"bytes"
	"context"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// exampleExampleComAddress is the address of example.com
const exampleExampleComAddress = "93.184.216.34"

// exampleClientAddress is the address used by the client
const exampleClientAddress = "130.192.91.211"

// exampleISPResolverAddress is the address used by the resolver.
const exampleISPResolverAddress = "130.192.3.24"

// exampleCensoredAddress is a censored IP address.
const exampleCensoredAddress = "10.10.34.35"

// exampleNewEnvironment creates a QA environment setting all possible options. We're going
// to use this QA environment in all the examples for this package.
func exampleNewEnvironment() *netemx.QAEnv {
	return netemx.NewQAEnv(
		netemx.QAEnvOptionDNSOverUDPResolvers("8.8.4.4", "9.9.9.9"),
		netemx.QAEnvOptionClientAddress(exampleClientAddress),
		netemx.QAEnvOptionISPResolverAddress(exampleISPResolverAddress),
		netemx.QAEnvOptionHTTPServer(exampleExampleComAddress, netemx.QAEnvDefaultHTTPHandler()),
		netemx.QAEnvOptionLogger(log.Log),
	)
}

// exampleAddRecordToAllResolvers shows how to add a DNS record to all the resolvers (i.e., including
// both the custom created resolvers and the ISP specific resolver).
func exampleAddRecordToAllResolvers(env *netemx.QAEnv) {
	env.AddRecordToAllResolvers(
		"example.com",
		"", // CNAME
		exampleExampleComAddress,
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
		exampleExampleComAddress,
	)

	// create a censored configuration for getaddrinfo
	env.ISPResolverConfig().AddRecord(
		"example.com",
		"",
		exampleCensoredAddress,
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
	env := netemx.NewQAEnv(
		netemx.QAEnvOptionNetStack(e1WhatsappNet, netemx.QAEnvNetStackTCPEcho(log.Log, 5222)),
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
