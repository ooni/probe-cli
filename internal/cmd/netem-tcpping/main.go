package main

import (
	"context"
	"flag"
	"math/rand"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/tcpping"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/qa"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func runCalibrationServer(ctx context.Context, server *netem.UNetStack, ready chan<- net.Listener) {
	buffer := make([]byte, 65535)
	_ = runtimex.Try1(rand.Read(buffer))

	addr := &net.TCPAddr{
		IP:   net.IPv4(10, 0, 0, 1),
		Port: 443,
		Zone: "",
	}
	listener := runtimex.Try1(server.ListenTCP("tcp", addr))
	ready <- listener

	for {
		conn, err := listener.Accept()
		if err != nil {
			break
		}
		conn.Close()
	}
}

func runCalibrationClient(ctx context.Context, client model.UnderlyingNetwork) {
	// create measurer for the dash experiment
	measurer := tcpping.NewExperimentMeasurer(tcpping.Config{})

	// create measurement to fill
	measurement := qa.NewMeasurement(measurer.ExperimentName(), measurer.ExperimentVersion())
	measurement.Input = "tcpconnect://10.0.0.1:443/"

	// create args for Run
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session: &mocks.Session{
			MockLogger: func() model.Logger {
				return log.Log
			},
		},
	}

	// measure inside a modified netxlite environment using stack
	var err error
	netxlite.WithCustomTProxy(client, func() {
		err = measurer.Run(ctx, args)
	})

	log.Infof("ERROR: %+v", err)
}

func main() {
	delay := flag.Duration("delay", 0, "LTR and RTL delay in millisecond")
	flag.Parse()

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logHandler}

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	gginfo := netem.NewStaticGetaddrinfo()
	cfg := netem.NewTLSMITMConfig()

	// create the client TCP/IP userspace stack
	client := netem.NewUNetStack("10.0.0.2", cfg, gginfo)

	// start capturing packets on the client side
	pcapClient := netem.NewPCAPDumper("tcpping.pcap", client)

	// create the server TCP/IP userspace stack
	server := netem.NewUNetStack("10.0.0.1", cfg, gginfo)

	// connect the two stacks using a link
	linkConfig := &netem.LinkConfig{
		LeftToRightDelay: *delay,
		LeftToRightPLR:   0,
		RightToLeftDelay: *delay,
		RightToLeftPLR:   0,
	}
	link := netem.NewLink(pcapClient, server, linkConfig)

	// start server in background and wait until it's listening
	serverListenerCh := make(chan net.Listener)
	go runCalibrationServer(ctx, server, serverListenerCh)
	listener := <-serverListenerCh

	// run client in foreground and measure speed
	runCalibrationClient(ctx, client)

	listener.Close()
	pcapClient.Close()
	server.Close()
	link.Close()
}
