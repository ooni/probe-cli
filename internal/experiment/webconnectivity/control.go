package webconnectivity

import (
	"context"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/webconnectivityalgo"
)

// Redirect to types defined inside the model package
type (
	ControlRequest           = model.THRequest
	ControlResponse          = model.THResponse
	ControlDNSResult         = model.THDNSResult
	ControlHTTPRequestResult = model.THHTTPRequestResult
	ControlTCPConnectResult  = model.THTCPConnectResult
)

// Control performs the control request and returns the response.
func Control(
	ctx context.Context, sess model.ExperimentSession,
	testhelpers []model.OOAPIService, creq ControlRequest) (ControlResponse, *model.OOAPIService, error) {
	sess.Logger().Infof("control for %s...", creq.HTTPRequest)
	out, idx, err := webconnectivityalgo.CallWebConnectivityTestHelper(ctx, &creq, testhelpers, sess)
	sess.Logger().Infof("control for %s... %+v", creq.HTTPRequest, model.ErrorToStringOrOK(err))
	if err != nil {
		// make sure error is wrapped
		//
		// IMPORTANT: here we create a new string error using `errors.New(err.Error())` because
		// `*httpclientx.ErrAllEndpointsFailed` implements the `Unwrap() []error` method which
		// would otherwise cause the `netxlite.NewTopLevelGenericErrWrapper` function to unwrap
		// as the first syscall error that occurred. So, without this error laundry passing through
		// strings, we would get the following:
		//
		//	connection_reset
		//
		// instead of
		//
		//	httpapi: all endpoints failed: [ connection_reset; connection_reset; connection_reset; connection_reset;]
		//
		// when running webconnectivity QA tests in the `webconnectivityqa` package.
		err = netxlite.NewTopLevelGenericErrWrapper(errors.New(err.Error()))
		return ControlResponse{}, nil, err
	}
	fillASNs(&out.DNS)
	runtimex.Assert(idx >= 0 && idx < len(testhelpers), "idx out of bounds")
	runtimex.Assert(out != nil, "out is nil")
	return *out, &testhelpers[idx], nil
}

// fillASNs fills the ASNs array of ControlDNSResult. For each Addr inside
// of the ControlDNSResult structure, we obtain the corresponding ASN.
//
// This is very useful to know what ASNs were the IP addresses returned by
// the control according to the probe's ASN database.
func fillASNs(dns *ControlDNSResult) {
	dns.ASNs = []int64{}
	for _, ip := range dns.Addrs {
		asn, _, _ := geoipx.LookupASN(ip)
		dns.ASNs = append(dns.ASNs, int64(asn))
	}
}
