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

func runCalibrationServer(ctx context.Context, server *netem.GvisorStack, ready, done chan any) {
	defer close(done)

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

func runCalibrationClient(ctx context.Context, client model.UnderlyingNetwork, done chan any) {
	defer close(done)

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
	bw := flag.Int64("bw", 0, "RTL bandwidth constraint")
	timeout := flag.Duration("timeout", 10*time.Second, "duration of the test")
	flag.Parse()

	logHandler := logx.NewHandlerWithDefaultSettings()
	logHandler.Emoji = true
	log.Log = &log.Logger{Level: log.InfoLevel, Handler: logHandler}

	gvisorCtx, gvisorCancel := context.WithCancel(context.Background())
	scCtx, scCancel := context.WithCancel(gvisorCtx)
	if *timeout > 0 {
		scCtx, scCancel = context.WithTimeout(gvisorCtx, *timeout)
	}
	defer scCancel()

	gginfo := netem.NewStaticGetaddrinfo()
	cfg := netem.NewTLSMITMConfig()

	// create the client TCP/IP userspace stack
	client := netem.NewGvisorStack("10.0.0.2", cfg, gginfo)
	left := &netem.NIC{
		Incoming: make(chan []byte, 4096),
		Name:     "client0",
		Outgoing: make(chan []byte, 4096),
	}
	client.Attach(gvisorCtx, left)

	// create the server TCP/IP userspace stack
	server := netem.NewGvisorStack("10.0.0.1", cfg, gginfo)
	right := &netem.NIC{
		Incoming: make(chan []byte, 4096),
		Name:     "server0",
		Outgoing: make(chan []byte, 4096),
	}
	server.Attach(gvisorCtx, right)

	// connect the two stacks using a link
	link := &netem.Link{
		DPI:                  &netem.DPINone{},
		Dump:                 false,
		Left:                 left,
		LeftToRightDelay:     *delay,
		LeftToRightBandwidth: 0,
		Right:                right,
		RightToLeftDelay:     *delay,
		RightToLeftBandwidth: netem.Bandwidth(*bw) * netem.KilobitsPerSecond,
	}
	link.Up(gvisorCtx)

	// start server in background and wait until it's listening
	serverReady := make(chan any)
	serverDone := make(chan any)
	go runCalibrationServer(scCtx, server, serverReady, serverDone)
	<-serverReady

	// run client in foreground and measure speed
	clientDone := make(chan any)
	runCalibrationClient(scCtx, client, clientDone)

	// wait for client and server to be done before shutting routing down
	<-serverDone
	<-clientDone
	gvisorCancel()
}
