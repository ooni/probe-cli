package oonimkall

//
// eXperimental OONI Run code.
//

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// OONIRunFetch fetches a given OONI run descriptor.
func (sess *Session) OONIRunFetch(ctx *Context, ID int64) (string, error) {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()

	clnt := sess.sessp.DefaultHTTPClient()

	// https://ams-pg-test.ooni.org/api/_/ooni_run/fetch/297500125102
	URL := &url.URL{
		Scheme:      "https",
		Opaque:      "",
		User:        nil,
		Host:        "ams-pg-test.ooni.org",
		Path:        fmt.Sprintf("/api/_/ooni_run/fetch/%d", ID),
		RawPath:     "",
		OmitHost:    false,
		ForceQuery:  false,
		RawQuery:    "",
		Fragment:    "",
		RawFragment: "",
	}

	req, err := http.NewRequestWithContext(ctx.ctx, "GET", URL.String(), nil)
	if err != nil {
		return "", err
	}

	resp, err := clnt.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("xoonirun: HTTP request failed")
	}

	rawResp, err := netxlite.ReadAllContext(ctx.ctx, resp.Body)
	if err != nil {
		return "", err
	}

	return string(rawResp), nil
}
