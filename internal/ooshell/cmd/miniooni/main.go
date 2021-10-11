package main

import (
	"context"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/ooshell"
)

func main() {
	ctx := context.Background()
	ooniHome, err := ooshell.OONIHome("miniooni")
	if err != nil {
		log.Fatalf("cannot determine OONI_HOME: %s", err.Error())
	}
	env := ooshell.NewEnvironment(ooniHome)
	env.NoCollector = true
	env.MaxRuntime = 25
	if err := env.RunExperiments(ctx, "custom", "ndt", "websteps"); err != nil {
		log.Fatalf("cannot run experiment: %s", err.Error())
	}
}
