package main

import (
	"context"
	"fmt"
	"log"
	"net"
)

func main() {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%s\n", listener.Addr().String())
	<-context.Background().Done()
}
