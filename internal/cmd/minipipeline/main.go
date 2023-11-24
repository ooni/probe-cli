package main

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/pipeline"
)

func main() {
	rawMeasurement := must.ReadFile(os.Args[1])
	var meas pipeline.CanonicalMeasurement
	must.UnmarshalJSON(rawMeasurement, &meas)

	container := minipipeline.NewWebObservationsContainer()
	container.CreateDNSLookupFailures(meas.TestKeys.Unwrap().Queries...)
	container.CreateKnownIPAddresses(meas.TestKeys.Unwrap().Queries...)
	container.CreateKnownTCPEndpoints(meas.TestKeys.Unwrap().TCPConnect...)
	container.NoteTLSHandshakeResults(meas.TestKeys.Unwrap().TLSHandshakes...)
	container.NoteHTTPRoundTripResults(meas.TestKeys.Unwrap().Requests...)
	container.NoteControlResults(meas.TestKeys.Unwrap().XControlRequest.Unwrap(), meas.TestKeys.Unwrap().Control.Unwrap())

	must.WriteFile("observation.json", must.MarshalJSON(container), 0600)

	analysis := &minipipeline.WebAnalysis{}
	analysis.ComputeDNSExperimentFailure(container)
	analysis.ComputeDNSTransactionsWithBogons(container)
	analysis.ComputeDNSTransactionsWithUnexpectedFailures(container)
	analysis.ComputeDNSPossiblyInvalidAddrs(container)
	analysis.ComputeTCPTransactionsWithUnexpectedTCPConnectFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedTLSHandshakeFailures(container)
	analysis.ComputeTCPTransactionsWithUnexpectedHTTPFailures(container)
	analysis.ComputeHTTPDiffBodyProportionFactor(container)
	analysis.ComputeHTTPDiffStatusCodeMatch(container)
	analysis.ComputeHTTPDiffUncommonHeadersIntersection(container)
	analysis.ComputeHTTPDiffTitleDifferentLongWords(container)

	must.WriteFile("analysis.json", must.MarshalJSON(analysis), 0600)
}
