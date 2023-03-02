package main

import (
	"context"
	"flag"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func runCalibrationServer(ctx context.Context, server *netem.UNetStack, ready chan any) {
	buffer := make([]byte, 65535)
	_ = runtimex.Try1(rand.Read(buffer))

	addr := &net.TCPAddr{
		IP:   net.IPv4(10, 0, 0, 1),
		Port: 443,
		Zone: "",
	}
	listener := runtimex.Try1(server.ListenTCP("tcp", addr))
	close(ready)

	conn := runtimex.Try1(listener.Accept())
	listener.Close()

	if deadline, okay := ctx.Deadline(); okay {
		conn.SetDeadline(deadline)
	}
	for {
		if _, err := conn.Write(buffer); err != nil {
			log.Warnf("runCalibrationServer: %s", err.Error())
			break
		}
	}
}

func runCalibrationClient(ctx context.Context, client model.UnderlyingNetwork) {
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	conn := runtimex.Try1(client.DialContext(ctx, 0, "tcp", "10.0.0.1:443"))
	defer conn.Close()

	if deadline, okay := ctx.Deadline(); okay {
		conn.SetDeadline(deadline)
	}

	buffer := make([]byte, 65535)

	var total int64
	t0 := time.Now()

	fmt.Printf("elapsed (s),total (byte),speed (Mbit/s)\n")
	for {
		count, err := conn.Read(buffer)
		if err != nil {
			log.Warnf("runCalibrationClient: %s", err.Error())
			return
		}
		total += int64(count)

		select {
		case <-ticker.C:
			elapsed := time.Since(t0).Seconds()
			speed := (float64(total*8) / elapsed) / (1000 * 1000)
			fmt.Printf("%f,%d,%f\n", elapsed, total, speed)
		case <-ctx.Done():
			return
		default:
			// nothing
		}
	}
}

func main() {
	delay := flag.Duration("delay", 0, "LTR and RTL delay in millisecond")
	plr := flag.Float64("plr", 0, "right-to-left packet loss rate")
	timeout := flag.Duration("timeout", 10*time.Second, "duration of the test")
	flag.Parse()

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logHandler}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

	gginfo := netem.NewStaticGetaddrinfo()
	cfg := netem.NewTLSMITMConfig()

	// create a backbone
	backbone := netem.NewBackbone()
	defer backbone.Close()

	const MTU = 64000

	// create the client TCP/IP userspace stack
	client := netem.NewUNetStack(MTU, "10.0.0.2", cfg, gginfo)

	// attach the client to the backbone
	clientLinkConfig := &netem.LinkConfig{
		Dump:             false,
		LeftToRightDelay: *delay,
		LeftToRightPLR:   0,
		RightToLeftDelay: *delay,
		RightToLeftPLR:   *plr,
	}
	backbone.AddStack(client, clientLinkConfig)

	// create the server TCP/IP userspace stack
	server := netem.NewUNetStack(MTU, "10.0.0.1", cfg, gginfo)

	// attach the server to the backbone.
	serverLinkConfig := &netem.LinkConfig{
		Dump:             false,
		LeftToRightPLR:   0,
		LeftToRightDelay: 0,
		RightToLeftDelay: 0,
		RightToLeftPLR:   0,
	}
	backbone.AddStack(server, serverLinkConfig)

	// start server in background and wait until it's listening
	serverReady := make(chan any)
	go runCalibrationServer(ctx, server, serverReady)
	<-serverReady

	// run client in foreground and measure speed
	go runCalibrationClient(ctx, client)

	<-ctx.Done()
}
