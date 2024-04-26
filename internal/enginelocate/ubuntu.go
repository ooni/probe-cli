package enginelocate

import (
	"context"
	"encoding/xml"
	"net"

	"github.com/ooni/probe-cli/v3/internal/httpx"
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
	// read the HTTP response body
	data, err := (&httpx.APIClientTemplate{
		BaseURL:    "https://geoip.ubuntu.com/",
		HTTPClient: httpClient,
		Logger:     logger,
		UserAgent:  userAgent,
	}).WithBodyLogging().Build().FetchResource(ctx, "/lookup")

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// parse the XML
	logger.Debugf("ubuntu: body: %s", string(data))
	var v ubuntuResponse
	err = xml.Unmarshal(data, &v)

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// make sure the IP addr is valid
	if net.ParseIP(v.IP) == nil {
		return model.DefaultProbeIP, ErrInvalidIPAddress
	}

	// handle the success case
	return v.IP, nil
}
