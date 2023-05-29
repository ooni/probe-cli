package netemx_test

import (
	"context"
	"net"
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/quic-go/quic-go/http3"
)

// Environment is the [netem] QA environment we use in this package.
//
// This struct provides a blueprint of how to write integration tests for
// other packages. For this reason, this code also includes support for DPI,
// even though this isn't strictly necessary for testing [netemx].
type Environment struct {
	// clientStack is the client stack to use.
	clientStack *netem.UNetStack

	// dnsServer is the DNS server.
	dnsServer *netem.DNSServer

	// dpi refers to the [netem.DPIEngine] we're using
	dpi *netem.DPIEngine

	// http3Server is the HTTP3 server.
	http3Server *http3.Server

	// httpsServer is the HTTPS server.
	httpsServer *http.Server

	// quicConn is the UDPLikeConn used by the HTTP/3 server.
	quicConn model.UDPLikeConn

	// topology is the topology we're using
	topology *netem.StarTopology
}

// NewEnvironment creates a new QA environment. This function
// calls [runtimex.PanicOnError] in case of failure.
func NewEnvironment() *Environment {
	// create a new star topology
	topology := runtimex.Try1(netem.NewStarTopology(model.DiscardLogger))

	// create server stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	serverStack := runtimex.Try1(topology.AddHost(
		"8.8.8.8", // server IP address
		"8.8.8.8", // default resolver address
		&netem.LinkConfig{},
	))

	// create configuration for DNS server
	dnsConfig := netem.NewDNSConfig()
	dnsConfig.AddRecord(
		"www.example.com",
		"private.example.com", // CNAME
		"10.0.17.1",
		"10.0.17.2",
		"10.0.17.3",
	)
	dnsConfig.AddRecord(
		"quad8.com",
		"", // CNAME
		"8.8.8.8",
	)

	// create DNS server using the serverStack
	dnsServer := runtimex.Try1(netem.NewDNSServer(
		model.DiscardLogger,
		serverStack,
		"8.8.8.8",
		dnsConfig,
	))

	// create HTTPS server using the serverStack
	tlsListener := runtimex.Try1(serverStack.ListenTCP("tcp", &net.TCPAddr{
		IP:   net.IPv4(8, 8, 8, 8),
		Port: 443,
		Zone: "",
	}))
	httpsServer := &http.Server{
		TLSConfig: serverStack.ServerTLSConfig(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`hello, world`))
		}),
	}
	go httpsServer.ServeTLS(tlsListener, "", "")

	// create HTTP3 server using the serverStack
	quicConn := runtimex.Try1(serverStack.ListenUDP("udp", &net.UDPAddr{
		IP:   net.IPv4(8, 8, 8, 8),
		Port: 443,
		Zone: "",
	}))
	http3Server := &http3.Server{
		TLSConfig: serverStack.ServerTLSConfig(),
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`hello, world`))
		}),
	}
	go http3Server.Serve(quicConn)

	// create a DPIEngine for implementing censorship
	dpi := netem.NewDPIEngine(model.DiscardLogger)

	// create client stack
	//
	// note: because the stack is created using topology.AddHost, we don't
	// need to call Close when done using it, since the topology will do that
	// for us when we call the topology's Close method.
	clientStack := runtimex.Try1(topology.AddHost(
		"10.0.0.14", // client IP address
		"8.8.8.8",   // default resolver address
		&netem.LinkConfig{
			DPIEngine: dpi,
		},
	))

	return &Environment{
		clientStack: clientStack,
		dnsServer:   dnsServer,
		dpi:         dpi,
		http3Server: http3Server,
		httpsServer: httpsServer,
		quicConn:    quicConn,
		topology:    topology,
	}
}

// DPIEngine returns the [netem.DPIEngine] we're using on the
// link between the client stack and the router. You can safely
// add new DPI rules from concurrent goroutines at any time.
func (e *Environment) DPIEngine() *netem.DPIEngine {
	return e.dpi
}

// Do executes the given function such that [netxlite] code uses the
// underlying clientStack rather than ordinary networking code.
func (e *Environment) Do(function func()) {
	netemx.WithCustomTProxy(e.clientStack, function)
}

// Close closes all the resources used by [Environment].
func (e *Environment) Close() error {
	e.dnsServer.Close()
	e.quicConn.Close()
	e.httpsServer.Close()
	e.http3Server.Close()
	e.topology.Close()
	return nil
}

// TestWithCustomTProxy ensures that we can use a [netem.UnderlyingNetwork] to
// hijack [netxlite] function calls to use TCP/IP in userspace.
func TestWithCustomTProxy(t *testing.T) {

	// Here we're testing that:
	//
	// 1. we can get the expected private answer for www.example.com, meaning that
	// we are using the userspace TCP/IP stack defined by [Environment].
	t.Run("we can hijack getaddrinfo lookups", func(t *testing.T) {
		env := NewEnvironment()
		defer env.Close()
		env.Do(func() {
			// create stdlib resolver, which will use the underlying client stack
			// GetaddrinfoLookupANY method for the DNS lookup
			reso := netxlite.NewStdlibResolver(model.DiscardLogger)

			// lookup the hostname
			ctx := context.Background()
			addrs, err := reso.LookupHost(ctx, "www.example.com")

			// verify that the result is okay
			if err != nil {
				t.Fatal(err)
			}
			expectAddrs := []string{
				"10.0.17.1",
				"10.0.17.2",
				"10.0.17.3",
			}
			if diff := cmp.Diff(expectAddrs, addrs); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// Here we're testing that:
	//
	// 1. we can get the expected answer for quad8.com;
	//
	// 2. we connect to the expected address;
	//
	// 3. we can successfully TLS handshake for quad8.com;
	//
	// 4. we obtain the expected webpage.
	//
	// If all of this works, it means we're using the userspace TCP/IP
	// stack exported by the [Environment] struct.
	t.Run("we can hijack HTTPS requests", func(t *testing.T) {
		env := NewEnvironment()
		defer env.Close()
		env.Do(func() {
			// create client, which will use the underlying client stack's
			// DialContext method to dial connections
			client := netxlite.NewHTTPClientStdlib(model.DiscardLogger)

			// create request using a domain that has been configured in the
			// [Environment] we're using as valid. Note that we're using https
			// and this will work because the client stack also controls the
			// default CA pool through the DefaultCertPool method.
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// make sure the status code and the body match
			if resp.StatusCode != 200 {
				t.Fatal("expected to see 200, got", resp.StatusCode)
			}
			expectBody := []byte(`hello, world`)
			gotBody, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expectBody, gotBody); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// Here we're testing that:
	//
	// 1. we can get the expected answer for quad8.com;
	//
	// 2. we can successfully QUIC handshake for quad8.com;
	//
	// 3. we obtain the expected webpage.
	//
	// If all of this works, it means we're using the userspace TCP/IP
	// stack exported by the [Environment] struct.
	t.Run("we can hijack HTTP3 requests", func(t *testing.T) {
		env := NewEnvironment()
		defer env.Close()
		env.Do(func() {
			// create an HTTP3 client
			txp := netxlite.NewHTTP3TransportStdlib(model.DiscardLogger)
			client := &http.Client{Transport: txp}

			// create the request; see above remarks for the HTTPS case
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer resp.Body.Close()

			// check the response and the body
			if resp.StatusCode != 200 {
				t.Fatal("expected to see 200, got", resp.StatusCode)
			}
			expectBody := []byte(`hello, world`)
			gotBody, err := netxlite.ReadAllContext(context.Background(), resp.Body)
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(expectBody, gotBody); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	// This is like the one where we test for HTTPS. The idea here is to
	// be sure that we can set DPI rules affecting the client stack.
	t.Run("we can configure DPI rules", func(t *testing.T) {
		env := NewEnvironment()
		defer env.Close()

		// create DPI rule blocking the quad8.com SNI with RST
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "quad8.com",
		})

		env.Do(func() {
			// create client, which will use the underlying client stack's
			// DialContext method to dial connections
			client := netxlite.NewHTTPClientStdlib(model.DiscardLogger)

			// create the request
			req, err := http.NewRequest("GET", "https://quad8.com/", nil)
			if err != nil {
				t.Fatal(err)
			}

			// issue the request
			resp, err := client.Do(req)

			// make sure we got a connection RST by peer error
			if err == nil || err.Error() != netxlite.FailureConnectionReset {
				t.Fatal("unexpected error", err)
			}
			if resp != nil {
				t.Fatal("expected nil response")
			}
		})
	})
}
