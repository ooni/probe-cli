package main

import (
	// commands

	_ "github.com/ooni/probe-cli/internal/cli/geoip"
	_ "github.com/ooni/probe-cli/internal/cli/info"
	_ "github.com/ooni/probe-cli/internal/cli/list"
	_ "github.com/ooni/probe-cli/internal/cli/nettest"
	_ "github.com/ooni/probe-cli/internal/cli/onboard"
	_ "github.com/ooni/probe-cli/internal/cli/reset"
	_ "github.com/ooni/probe-cli/internal/cli/run"
	_ "github.com/ooni/probe-cli/internal/cli/show"
	_ "github.com/ooni/probe-cli/internal/cli/upload"
	_ "github.com/ooni/probe-cli/internal/cli/version"
	"github.com/ooni/probe-cli/internal/crashreport"

	"github.com/ooni/probe-cli/internal/cli/app"
)

func main() {
	crashreport.CapturePanicAndWait(app.Run, nil)
}
