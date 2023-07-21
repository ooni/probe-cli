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
//
// The ID argument is the unique identifier of the OONI Run link. For example, in:
//
//	https://api.ooni.io/api/_/ooni_run/fetch/297500125102
//
// The OONI Run link ID is 297500125102.
//
// Warning: this API is currently experimental and we only expose it to facilitate
// developing OONI Run v2. Do not use this API in production.
func (sess *Session) OONIRunFetch(ctx *Context, ID int64) (string, error) {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()

	// TODO(bassosimone): this code should be changed to use the probeservices.Client
	// rather than using an hardcoded URL once we switch to production code. Until then,
	// we are going to use the test backend server.

	// For example: https://ams-pg-test.ooni.org/api/_/ooni_run/fetch/297500125102
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

	return sess.ooniRunFetchWithURLLocked(ctx, URL)
}

func (sess *Session) ooniRunFetchWithURLLocked(ctx *Context, URL *url.URL) (string, error) {
	clnt := sess.sessp.DefaultHTTPClient()

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
