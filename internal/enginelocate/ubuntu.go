package enginelocate

import (
	"context"
	"encoding/xml"
	"net"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type ubuntuResponse struct {
	XMLName xml.Name `xml:"Response"`
	IP      string   `xml:"Ip"`
}

func ubuntuIPLookup(
	ctx context.Context,
	httpClient model.HTTPClient,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	// read the HTTP response and parse as XML
	resp, err := httpclientx.GetXML[*ubuntuResponse](ctx, "https://geoip.ubuntu.com/lookup", &httpclientx.Config{
		Authorization: "", // not needed
		Client:        httpClient,
		Logger:        logger,
		UserAgent:     userAgent,
	})

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// make sure the IP addr is valid
	if net.ParseIP(resp.IP) == nil {
		return model.DefaultProbeIP, ErrInvalidIPAddress
	}

	// handle the success case
	return resp.IP, nil
}
