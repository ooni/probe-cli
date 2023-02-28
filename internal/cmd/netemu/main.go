package main

//
// Test client for ./internal/netem. Will be removed before merging.
//

import (
	"context"
	"log"
	"net"
	"net/http"
	"net/url"
	"time"

	apexlog "github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/dash"
	"github.com/ooni/probe-cli/v3/internal/humanize"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func runServerUDP(ctx context.Context, ns *netem.GvisorStack, ready chan any) {
	addr := &net.UDPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 4096,
		Zone: "",
	}
	pconn, err := ns.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	close(ready)
	for ctx.Err() == nil {
		buffer := make([]byte, 1024)
		count, addr, err := pconn.ReadFrom(buffer)
		if err != nil {
			log.Printf("pconn.ReadFrom: %s", err.Error())
			continue
		}
		buffer = buffer[:count]
		log.Printf("server: got %d bytes from %s", count, addr.String())
		_, err = pconn.WriteTo(buffer, addr)
		if err != nil {
			log.Printf("pconn.WriteTo: %s", err.Error())
			continue
		}
	}
}

func runServerTCP(ctx context.Context, ns *netem.GvisorStack, ready chan any) {
	addr := &net.TCPAddr{
		IP:   net.IPv4(10, 17, 17, 1),
		Port: 4096,
		Zone: "",
	}
	listener, err := ns.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}
	close(ready)
	conn, err := listener.Accept()
	if err != nil {
		log.Fatal(err)
	}
	for ctx.Err() == nil {
		buffer := make([]byte, 1024)
		count, err := conn.Read(buffer)
		if err != nil {
			log.Printf("conn.Read: %s", err.Error())
			continue
		}
		buffer = buffer[:count]
		log.Printf("server: got %d bytes from peer", count)
		_, err = conn.Write(buffer)
		if err != nil {
			log.Printf("conn.Write: %s", err.Error())
			continue
		}
	}
}

func runClientUDP(ctx context.Context, ns *netem.GvisorStack) {
	addr := &net.UDPAddr{
		IP:   net.IPv4(10, 17, 17, 34),
		Port: 0,
		Zone: "",
	}
	pconn, err := ns.ListenUDP("udp", addr)
	if err != nil {
		log.Fatal(err)
	}
	for ctx.Err() == nil {
		destAddr := &net.UDPAddr{
			IP:   net.IPv4(10, 17, 17, 1),
			Port: 4096,
			Zone: "",
		}
		message := []byte(string("ciao"))
		_, err := pconn.WriteTo(message, destAddr)
		if err != nil {
			log.Printf("pconn.WriteTo: %s", err.Error())
			continue
		}
		buffer := make([]byte, 1024)
		count, senderAddr, err := pconn.ReadFrom(buffer)
		if err != nil {
			log.Printf("pconn.ReadFrom: %s", err.Error())
			continue
		}
		log.Printf("client: got %d bytes from %s", count, senderAddr.String())
	}
}

func runClientTCP(ctx context.Context, ns *netem.GvisorStack) {
	conn, err := ns.DialContext(ctx, 10*time.Second, "tcp", "10.17.17.1:4096")
	if err != nil {
		log.Fatal(err)
	}
	for ctx.Err() == nil {
		message := []byte(string("ciao"))
		_, err := conn.Write(message)
		if err != nil {
			log.Printf("conn.Write: %s", err.Error())
			continue
		}
		buffer := make([]byte, 1024)
		count, err := conn.Read(buffer)
		if err != nil {
			log.Printf("pconn.ReadFrom: %s", err.Error())
			continue
		}
		buffer = buffer[:count]
		log.Printf("client: got %d bytes from peer", count)
	}
}

func sendSingleDashRequest(ctx context.Context, client *netem.GvisorStack, domain string) {
	netxlite.WithCustomTProxy(client, func() {
		URL := &url.URL{
			Scheme: "https",
			Host:   domain,
			Path:   "/dash/download/26214400", // 25 MiB
		}
		client := netxlite.NewHTTPClientStdlib(model.DiscardLogger)
		req := runtimex.Try1(http.NewRequestWithContext(ctx, "GET", URL.String(), nil))
		resp := runtimex.Try1(client.Do(req))
		log.Printf("%+v", resp)
		defer resp.Body.Close()
		var total int64
		ticker := time.NewTicker(500 * time.Millisecond)
		defer ticker.Stop()
		t0 := time.Now()
		for {
			buffer := make([]byte, 8000)
			count, err := resp.Body.Read(buffer)
			if err != nil {
				log.Printf("resp.Body.Read: %s", err.Error())
				break
			}
			total += int64(count)
			select {
			case t1 := <-ticker.C:
				speed := float64(8*total) / t1.Sub(t0).Seconds()
				log.Printf("speed: %s", humanize.SI(speed, "bit/s"))
			default:
			}
		}
	})
}

func runTheDashExperiment(ctx context.Context, client *netem.GvisorStack) {
	netxlite.WithCustomTProxy(client, func() {
		measurer := dash.NewExperimentMeasurer(dash.Config{})
		measurement := &model.Measurement{}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(apexlog.Log),
			Measurement: measurement,
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return apexlog.Log
				},
				MockUserAgent: func() string {
					return "miniooni/" + version.Version
				},
			},
		}
		err := measurer.Run(ctx, args)
		log.Printf("ERROR: %+v", err)
	})
}

func withDash() {
	// TODO(bassosimone): creating servers manually like this is obviously
	// difficult and error prone; we need some higher-level abstraction

	// create overarching context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// fill DNS information
	const dashServerAddr = "130.192.182.100"
	const locateServerAddr = "130.192.182.101"
	gginfo := netem.NewStaticGetaddrinfo()
	gginfo.AddStaticEntry(netem.DefaultMLabLocateDASHDomain, &netem.StaticGetaddrinfoEntry{
		Addresses: []string{
			dashServerAddr,
		},
		CNAME: "",
	})
	gginfo.AddStaticEntry(netem.DefaultMLabLocateDomain, &netem.StaticGetaddrinfoEntry{
		Addresses: []string{
			locateServerAddr,
		},
		CNAME: "",
	})

	// create configuration for performing TLS MITM
	mitmCfg := netem.NewTLSMITMConfig()

	// create a network backbone
	backbone := netem.NewBackbone()

	// create a client stack
	client := netem.NewGvisorStack("130.192.91.211", mitmCfg, gginfo)
	backbone.AddClient(ctx, client, netem.NewLinkMedium) // change link here to change perf

	// create a server stack
	server := netem.NewGvisorStack(dashServerAddr, mitmCfg, gginfo)
	backbone.AddServer(ctx, server, netem.NewLinkFastest, &netem.DPINone{})

	// run the locatev2 server using the server stack
	server2 := netem.NewGvisorStack(locateServerAddr, mitmCfg, gginfo)
	backbone.AddServer(ctx, server2, netem.NewLinkFastest, &netem.DPINone{})
	locateServer := netem.NewMLabLocateServer(server2, mitmCfg, locateServerAddr, &netem.MLabLocateConfig{
		DASH: netem.NewMLabLocateConfigDASH(),
	})
	defer locateServer.Stop()

	// run the DASH server using the server stack
	dashServer := netem.NewDASHServer(server, mitmCfg, dashServerAddr)
	defer dashServer.Stop()

	runTheDashExperiment(ctx, client)
}

func main() {
	withDash()
}
