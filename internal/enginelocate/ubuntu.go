package enginelocate

import (
	"context"
	"encoding/xml"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type ubuntuResponse struct {
	XMLName xml.Name `xml:"Response"`
	IP      string   `xml:"Ip"`
}

func ubuntuIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	// read the HTTP response and parse as XML
	resp, err := httpclientx.GetXML[*ubuntuResponse](
		ctx,
		"https://geoip.ubuntu.com/lookup",
		httpClient,
		logger,
		userAgent,
	)

	// handle the error case
	if err != nil {
		return model.DefaultProbeIP, err
	}

	// handle the success case
	return resp.IP, nil
}
