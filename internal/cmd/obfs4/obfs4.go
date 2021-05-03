package main

import (
	"context"
	"flag"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/tunnel"
)

func main() {
	address := flag.String("address", "", "Bridge address")
	cert := flag.String("cert", "", "bridge cert")
	iatmode := flag.String("iat-mode", "", "bridge iat-mode")
	flag.Parse()
	log.SetLevel(log.DebugLevel)
	obfs4 := &tunnel.OBFS4{
		Address: *address,
		Cert:    *cert,
		DataDir: "obfs4xx",
		IATMode: *iatmode,
		Logger:  log.Log,
	}
	obfs4.Start(context.Background())
	<-context.Background().Done()
}
