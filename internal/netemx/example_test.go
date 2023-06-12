package netemx_test

import (
	"fmt"
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func Example() {
	// exampleComAddress is the IP address used for the example.com domain
	const exampleComAddress = "93.184.216.34"

	// create common DNS configuration for clients and servers
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord(
		"example.com",
		"netem.example.com", // CNAME
		exampleComAddress,
	)

	// describe the client side of the topology
	clientConfig := &netemx.ClientConfig{
		ClientAddr:   "", // use the default value
		DNSConfig:    dnsConfig,
		ResolverAddr: "", // use the default value
	}

	// describe the server side of the topology
	serversConfig := &netemx.ServersConfig{
		DNSConfig:    dnsConfig,
		ResolverAddr: "", // use the default value
		Servers: []netemx.ConfigServerStack{{
			ServerAddr: exampleComAddress,
			HTTPServers: []netemx.ConfigHTTPServer{{
				Port:    443,
				QUIC:    false,
				Handler: nil, // use the default handler
			}},
		}},
	}

	// create the environment for running the tests
	env := netemx.NewEnvironment(clientConfig, serversConfig)

	// make sure we close all the resources when we're done
	defer env.Close()

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
