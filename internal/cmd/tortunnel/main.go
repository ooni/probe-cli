package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/tortunnel"
)

func main() {
	config := &tortunnel.Config{
		Logger: log.Log,
	}
	ctx := context.Background()
	tunnel, err := tortunnel.Start(ctx, config)
	if err != nil {
		log.Fatalf("failure: %s", err.Error())
	}
	tunnel.Stop()
}
