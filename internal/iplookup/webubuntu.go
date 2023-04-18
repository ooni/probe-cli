package iplookup

//
// IP lookup using Ubuntu
//

import (
	"context"
	"encoding/xml"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ubuntuResponse is the XML response returned by ubuntu.
type ubuntuResponse struct {
	XMLName xml.Name `xml:"Response"`
	IP      string   `xml:"Ip"`
}

// ubuntuWebLookup implements fallback.Service
type ubuntuWebLookup struct {
	client *Client
}

// newUbuntuWebLookup creates a new [ubuntuWebLookup] instance.
func newUbuntuWebLookup(client *Client) *ubuntuWebLookup {
	return &ubuntuWebLookup{client}
}

var _ fallback.Service[model.AddressFamily, string] = &ubuntuWebLookup{}

// lookupUbuntu performs the lookup using ubuntu.
func (svc *ubuntuWebLookup) Run(ctx context.Context, family model.AddressFamily) (string, error) {
	// create HTTP request
	const URL = "https://geoip.ubuntu.com/lookup"
	req := runtimex.Try1(http.NewRequestWithContext(ctx, http.MethodGet, URL, nil))
	req.Header.Set("User-Agent", model.HTTPHeaderUserAgent)

	// send request and get response body
	data, err := svc.client.httpDo(req, family)
	if err != nil {
		return "", err
	}

	// parse the response body to obtain the IP address
	var v ubuntuResponse
	err = xml.Unmarshal(data, &v)
	if err != nil {
		return "", err
	}
	return v.IP, nil
}

// URL implements fallback.Service
func (cl *ubuntuWebLookup) URL() string {
	return "iplookup+web://ubuntu/"
}
