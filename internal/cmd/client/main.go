package main

import (
	"context"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	dialer := &net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	log.Print(err)
	conn.Close()
}
