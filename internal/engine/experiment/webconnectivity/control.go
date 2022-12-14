package webconnectivity

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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
	seqCaller := httpapi.NewSequenceCaller(
		ooapi.NewDescriptorTH(&creq),
		httpapi.NewEndpointList(sess.DefaultHTTPClient(), sess.Logger(), sess.UserAgent(), testhelpers...)...,
	)
	sess.Logger().Infof("control for %s...", creq.HTTPRequest)
	var out ControlResponse
	idx, err := seqCaller.CallWithJSONResponse(ctx, &out)
	sess.Logger().Infof("control for %s... %+v", creq.HTTPRequest, model.ErrorToStringOrOK(err))
	if err != nil {
		// make sure error is wrapped
		err = netxlite.NewTopLevelGenericErrWrapper(err)
		return ControlResponse{}, nil, err
	}
	fillASNs(&out.DNS)
	runtimex.Assert(idx >= 0 && idx < len(testhelpers), "idx out of bounds")
	return out, &testhelpers[idx], nil
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
