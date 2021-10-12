package nettests

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/dnscheck"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/run"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSCheck nettest implementation.
type DNSCheck struct{}

var dnsCheckDefaultInput []string

func dnsCheckMustMakeInput(input *run.StructuredInput) string {
	data, err := json.Marshal(input)
	runtimex.PanicOnError(err, "json.Marshal failed")
	return string(data)
}

func init() {
	// The following code just adds a minimal set of URLs to
	// test using DNSCheck, so we start exposing it.
	//
	// TODO(bassosimone):
	//
	// 1. we should be getting input from the backend instead of
	// having an hardcoded list of inputs here.
	//
	// 2. we should modify dnscheck to accept http3://... as a
	// shortcut for https://... with h3. If we don't do that, we
	// are stuck with the h3 results hiding h2 results in OONI
	// Explorer because they use the same URL.
	//
	// 3. it seems we have the problem that dnscheck results
	// appear as the `run` nettest in `ooniprobe list <ID>` because
	// dnscheck is run using the `run` functionality.
	dnsCheckDefaultInput = append(dnsCheckDefaultInput, dnsCheckMustMakeInput(
		&run.StructuredInput{
			DNSCheck: dnscheck.Config{},
			Name:     "dnscheck",
			Input:    "https://dns.google/dns-query",
		}))
	dnsCheckDefaultInput = append(dnsCheckDefaultInput, dnsCheckMustMakeInput(
		&run.StructuredInput{
			DNSCheck: dnscheck.Config{},
			Name:     "dnscheck",
			Input:    "https://cloudflare-dns.com/dns-query",
		}))
}

// Run starts the nettest.
func (n DNSCheck) Run(ctl *Controller) error {
	builder, err := ctl.Session.NewExperimentBuilder("dnscheck")
	if err != nil {
		return err
	}
	return ctl.Run(builder, dnsCheckDefaultInput)
}
