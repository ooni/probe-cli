package webconnectivitylte

import (
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func (tk *TestKeys) analysisClassic(logger model.Logger) {
	// Since we run after all tasks have completed (or so we assume) we're
	// not going to use any form of locking here.

	// 1. produce observations using the minipipeline
	container := minipipeline.NewWebObservationsContainer()
	container.IngestDNSLookupEvents(tk.Queries...)
	container.IngestTCPConnectEvents(tk.TCPConnect...)
	container.IngestTLSHandshakeEvents(tk.TLSHandshakes...)
	container.IngestHTTPRoundTripEvents(tk.Requests...)

	// be defensive in case the control request or control are not defined
	if tk.ControlRequest != nil && tk.Control != nil {
		// Implementation note: the only error that can happen here is when the input
		// doesn't parse as a URL, which should have triggered previous errors
		runtimex.Try0(container.IngestControlMessages(tk.ControlRequest, tk.Control))
	}

	// 2. filter observations to only include results collected by the
	// system resolver, which approximates v0.4's results
	classic := minipipeline.ClassicFilter(container)

	// 3. run the classic analysis algorithm
	tk.analysisClassicOld(logger, classic)
}
