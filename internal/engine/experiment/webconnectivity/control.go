package webconnectivity

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
	thAddr string, creq ControlRequest) (out ControlResponse, err error) {
	clnt := &httpx.APIClientTemplate{
		BaseURL:    thAddr,
		HTTPClient: sess.DefaultHTTPClient(),
		Logger:     sess.Logger(),
		UserAgent:  sess.UserAgent(),
	}
	sess.Logger().Infof("control for %s...", creq.HTTPRequest)
	// make sure error is wrapped
	err = clnt.WithBodyLogging().Build().PostJSON(ctx, "/", creq, &out)
	if err != nil {
		err = netxlite.NewTopLevelGenericErrWrapper(err)
	}
	sess.Logger().Infof("control for %s... %+v", creq.HTTPRequest, model.ErrorToStringOrOK(err))
	fillASNs(&out.DNS)
	return
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
