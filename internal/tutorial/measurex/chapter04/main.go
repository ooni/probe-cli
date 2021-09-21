package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	sni := flag.String("sni", "dns.google", "domain to resolver")
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := mx.TLSConnectAndHandshake(ctx, *address, &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
	})
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
