package websteps

import (
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/websteps"
)

type (
	URLMeasurement           = websteps.URLMeasurement
	DNSMeasurement           = websteps.DNSMeasurement
	EndpointMeasurement      = websteps.EndpointMeasurement
	TCPConnectMeasurement    = websteps.TCPConnectMeasurement
	HTTPRoundTripMeasurement = websteps.HTTPRoundTripMeasurement
	TLSHandshakeMeasurement  = websteps.TLSHandshakeMeasurement
	HTTPRequestMeasurement   = websteps.HTTPRequestMeasurement
	HTTPResponseMeasurement  = websteps.HTTPResponseMeasurement
	RoundTrip                = websteps.RoundTrip
)
