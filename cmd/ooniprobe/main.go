package main

import (
	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/app"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/autorun"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/geoip"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/info"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/list"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/onboard"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/reset"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/rm"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/run"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/show"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/upload"
	_ "github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/cli/version"
	"github.com/ooni/probe-cli/v3/cmd/ooniprobe/internal/crashreport"
)

func main() {
	if err, _ := crashreport.CapturePanic(app.Run, nil); err != nil {
		log.WithError(err.(error)).Error("panic in app.Run")
		crashreport.Wait()
	}
}
