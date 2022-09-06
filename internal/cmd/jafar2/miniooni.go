package main

import (
	"context"
	"net"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// miniooniConnReader reads from miniooni's [conn] and writes to [ch]. This goroutine will
// return if reading [conn] returns any error or [ctx] is done.
func miniooniConnReader(ctx context.Context, conn net.Conn, ch chan<- []byte) {
	for {
		rawPacket, err := netxlite.ReadSimpleFrame(conn)
		if err != nil {
			log.Warnf("miniooniConnRead: %s", err.Error())
			return
		}
		select {
		case ch <- rawPacket:
		case <-ctx.Done():
			return
		}
	}
}

// miniooniConnWriter reads from [ch] and writes to miniooni's [conn]. This goroutine will
// return if reading [conn] returns any error or [ctx] is done.
func miniooniConnWriter(ctx context.Context, ch <-chan []byte, conn net.Conn) {
	for {
		select {
		case rawPacket := <-ch:
			if err := netxlite.WriteSimpleFrame(conn, rawPacket); err != nil {
				log.Warnf("miniooniConnWrite: %s", err.Error())
				return
			}
		case <-ctx.Done():
			return
		}
	}
}
