package main

import (
	// commands
	_ "github.com/openobservatory/gooni/internal/cli/info"
	_ "github.com/openobservatory/gooni/internal/cli/list"
	_ "github.com/openobservatory/gooni/internal/cli/nettest"
	_ "github.com/openobservatory/gooni/internal/cli/run"
	_ "github.com/openobservatory/gooni/internal/cli/show"
	_ "github.com/openobservatory/gooni/internal/cli/upload"
	_ "github.com/openobservatory/gooni/internal/cli/version"
	"github.com/openobservatory/gooni/internal/util"

	"github.com/openobservatory/gooni/internal/cli/app"
)

func main() {
	err := app.Run()
	if err == nil {
		return
	}
	util.Fatal(err)
}
