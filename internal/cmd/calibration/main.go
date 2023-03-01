package main

import (
	"context"
	"fmt"
	"math/rand"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func runServer(ctx context.Context, server *netem.GvisorStack, done chan any) {
	buffer := make([]byte, 65535)
	_ = runtimex.Try1(rand.Read(buffer))
	addr := &net.TCPAddr{
		IP:   net.IPv4(10, 0, 0, 1),
		Port: 443,
		Zone: "",
	}
	listener := runtimex.Try1(server.ListenTCP("tcp", addr))
	defer listener.Close()
	close(done)
	for {
		conn := runtimex.Try1(listener.Accept())
		for {
			if _, err := conn.Write(buffer); err != nil {
				break
			}
		}
	}
}

func main() {
	ctx := context.Background()

	gginfo := netem.NewStaticGetaddrinfo()
	cfg := netem.NewTLSMITMConfig()

	client := netem.NewGvisorStack("10.0.0.2", cfg, gginfo)
	left := &netem.NIC{
		Incoming:      make(chan []byte, 4096),
		Name:          "client0",
		Outgoing:      make(chan []byte, 4096),
		RecvBandwidth: 2000 * netem.KilobitsPerSecond,
		SendBandwidth: 2000 * netem.KilobitsPerSecond,
	}
	client.Attach(ctx, left)

	server := netem.NewGvisorStack("10.0.0.1", cfg, gginfo)
	right := &netem.NIC{
		Incoming:      make(chan []byte, 4096),
		Name:          "server0",
		Outgoing:      make(chan []byte, 4096),
		RecvBandwidth: 0,
		SendBandwidth: 0,
	}
	server.Attach(ctx, right)

	link := &netem.Link{
		DPI:              &netem.DPINone{},
		Dump:             false,
		Left:             left,
		LeftToRightDelay: 0 * time.Millisecond,
		Right:            right,
		RightToLeftDelay: 0 * time.Millisecond,
	}
	link.Up(ctx)

	done := make(chan any)
	go runServer(ctx, server, done)
	<-done

	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
	conn := runtimex.Try1(client.DialContext(ctx, 0, "tcp", "10.0.0.1:443"))
	buffer := make([]byte, 65535)
	var total int64
	t0 := time.Now()
	fmt.Printf("elapsed (s),total (byte),speed (Mbit/s)\n")
	for {
		count := runtimex.Try1(conn.Read(buffer))
		total += int64(count)
		select {
		case <-ticker.C:
			elapsed := time.Since(t0).Seconds()
			speed := (float64(total*8) / elapsed) / (1000 * 1000)
			fmt.Printf("%f,%d,%f\n", elapsed, total, speed)
		default:
			// nothing
		}
	}
}
