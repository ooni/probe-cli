package iplookup

//
// Code to resolve the IP address using Ubuntu
//

import (
	"context"
	"encoding/xml"
	"net"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ubuntuResponse is the XML response returned by ubuntu.
type ubuntuResponse struct {
	XMLName xml.Name `xml:"Response"`
	IP      string   `xml:"Ip"`
}

// lookupUbuntu performs the lookup using ubuntu.
func (c *Client) lookupUbuntu(ctx context.Context, family Family) (string, error) {
	// make sure we eventually time out
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// create HTTP request
	const URL = "https://geoip.ubuntu.com/lookup"
	req := runtimex.Try1(http.NewRequestWithContext(ctx, http.MethodGet, URL, nil))
	req.Header.Set("User-Agent", model.HTTPHeaderUserAgent)

	// send request and get response body
	data, err := c.httpDo(req, family)
	if err != nil {
		return "", err
	}

	// parse the response body to obtain the IP address
	var v ubuntuResponse
	err = xml.Unmarshal(data, &v)
	if err != nil {
		return "", err
	}

	// make sure the IP address is valid
	if net.ParseIP(v.IP) == nil {
		return "", ErrInvalidIPAddress
	}

	return v.IP, nil
}
