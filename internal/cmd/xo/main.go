package main

import (
	"context"
	"fmt"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/loader"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	log.SetLevel(log.DebugLevel)

	keyValueStore := runtimex.Try1(kvstore.NewFS("xodir"))
	txp := netxlite.NewHTTPTransportStdlib(log.Log)
	client := loader.NewClient("api.ooni.io", log.Log, keyValueStore, txp)

	pi := loader.ProbeInfo{
		Charging:        false,
		OnWiFi:          false,
		Platform:        "linux",
		ProbeASN:        "AS137",
		ProbeCC:         "IT",
		RunType:         "manual",
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
	}

	/*
		query := &loader.WebConnectivityQuery{
			ProbeInfo:     pi,
			CategoryCodes: []string{"NEWS"},
		}
	*/

	ctx := context.Background()

	//spec := runtimex.Try1(client.LoadWebConnectivity(ctx, query))
	//spec := runtimex.Try1(client.LoadRiseupVPN(ctx, &pi))
	spec := runtimex.Try1(client.LoadTor(ctx, &pi))

	fmt.Printf("%s\n", string(must.MarshalJSON(spec)))
}
