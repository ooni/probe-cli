package main

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/app"
	_ "github.com/ooni/probe-cli/internal/cli/geoip"
	_ "github.com/ooni/probe-cli/internal/cli/info"
	_ "github.com/ooni/probe-cli/internal/cli/list"
	_ "github.com/ooni/probe-cli/internal/cli/onboard"
	_ "github.com/ooni/probe-cli/internal/cli/reset"
	_ "github.com/ooni/probe-cli/internal/cli/rm"
	_ "github.com/ooni/probe-cli/internal/cli/run"
	_ "github.com/ooni/probe-cli/internal/cli/show"
	_ "github.com/ooni/probe-cli/internal/cli/upload"
	_ "github.com/ooni/probe-cli/internal/cli/version"
	"github.com/ooni/probe-cli/internal/crashreport"
)

func main() {
	if err, _ := crashreport.CapturePanic(app.Run, nil); err != nil {
		log.WithError(err.(error)).Error("panic in app.Run")
		crashreport.Wait()
	}
}
