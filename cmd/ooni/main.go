package main

import (
	// commands
	"github.com/apex/log"
	"github.com/getsentry/raven-go"

	_ "github.com/ooni/probe-cli/internal/cli/geoip"
	_ "github.com/ooni/probe-cli/internal/cli/info"
	_ "github.com/ooni/probe-cli/internal/cli/list"
	_ "github.com/ooni/probe-cli/internal/cli/nettest"
	_ "github.com/ooni/probe-cli/internal/cli/onboard"
	_ "github.com/ooni/probe-cli/internal/cli/run"
	_ "github.com/ooni/probe-cli/internal/cli/show"
	_ "github.com/ooni/probe-cli/internal/cli/upload"
	_ "github.com/ooni/probe-cli/internal/cli/version"

	"github.com/ooni/probe-cli/internal/cli/app"
)

func main() {
	raven.SetDSN("https://cb4510e090f64382ac371040c19b2258:8448daeebfa643c289ef398f8645980b@sentry.io/1234954")

	err := app.Run()
	if err == nil {
		return
	}
	log.WithError(err).Fatal("main exit")
}
