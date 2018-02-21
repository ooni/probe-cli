package main

import (
	// commands
	"github.com/apex/log"

	_ "github.com/openobservatory/gooni/internal/cli/info"
	_ "github.com/openobservatory/gooni/internal/cli/list"
	_ "github.com/openobservatory/gooni/internal/cli/nettest"
	_ "github.com/openobservatory/gooni/internal/cli/run"
	_ "github.com/openobservatory/gooni/internal/cli/show"
	_ "github.com/openobservatory/gooni/internal/cli/upload"
	_ "github.com/openobservatory/gooni/internal/cli/version"

	"github.com/openobservatory/gooni/internal/cli/app"
)

func main() {
	err := app.Run()
	if err == nil {
		return
	}
	log.WithError(err).Fatal("main exit")
}
