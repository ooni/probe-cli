package oohelperd_test

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/oohelperd"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TestQAEnableDisableQUIC ensures that we can enable and disable QUIC.
func TestQAEnableDisableQUIC(t *testing.T) {
	// testcase is a test case for this function
	type testcase struct {
		name       string
		enableQUIC optional.Value[bool]
	}

	cases := []testcase{{
		name:       "with the default settings",
		enableQUIC: optional.None[bool](),
	}, {
		name:       "with explicit false",
		enableQUIC: optional.Some(false),
	}, {
		name:       "with explicit true",
		enableQUIC: optional.Some(true),
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create a new testing scenario
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			// create a new handler
			handler := oohelperd.NewHandler(
				log.Log,
				&netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack}},
			)

			// optionally and conditionally enable QUIC
			if !tc.enableQUIC.IsNone() {
				handler.EnableQUIC = tc.enableQUIC.Unwrap()
			}

			// create request body
			reqbody := &model.THRequest{
				HTTPRequest: "https://www.example.com/",
				HTTPRequestHeaders: map[string][]string{
					"Accept-Language": {model.HTTPHeaderAcceptLanguage},
					"Accept":          {model.HTTPHeaderAccept},
					"User-Agent":      {model.HTTPHeaderUserAgent},
				},
				TCPConnect:   []string{netemx.AddressWwwExampleCom},
				XQUICEnabled: true,
			}

			// create request
			req := runtimex.Try1(http.NewRequest(
				"POST",
				"http://127.0.0.1:8080/",
				bytes.NewReader(must.MarshalJSON(reqbody)),
			))

			// create response recorder
			resprec := httptest.NewRecorder()

			// invoke the handler
			handler.ServeHTTP(resprec, req)

			// get the response
			resp := resprec.Result()
			defer resp.Body.Close()

			// make sure the status code indicates success
			if resp.StatusCode != 200 {
				t.Fatal("expected 200 Ok")
			}

			// make sure the content-type is OK
			if v := resp.Header.Get("Content-Type"); v != "application/json" {
				t.Fatal("unexpected content-type", v)
			}

			// read the response body
			respbody := runtimex.Try1(netxlite.ReadAllContext(context.Background(), resp.Body))

			// parse the response body
			var jsonresp model.THResponse
			must.UnmarshalJSON(respbody, &jsonresp)

			// check whether we have an HTTP3 response
			switch {
			case !tc.enableQUIC.IsNone() && tc.enableQUIC.Unwrap() && jsonresp.HTTP3Request != nil:
				// all good: we have QUIC enabled and we get an HTTP/3 response

			case (tc.enableQUIC.IsNone() || tc.enableQUIC.Unwrap() == false) && jsonresp.HTTP3Request == nil:
				// all good: either default behavior or QUIC not enabled and not HTTP/3 response

			default:
				t.Fatalf(
					"tc.enableQUIC.IsNone() = %v, tc.enableQUIC.UnwrapOr(false) = %v, jsonresp.HTTP3Request = %v",
					tc.enableQUIC.IsNone(),
					tc.enableQUIC.UnwrapOr(false),
					jsonresp.HTTP3Request,
				)
			}

			// check whether we have QUIC handshakes
			switch {
			case !tc.enableQUIC.IsNone() && tc.enableQUIC.Unwrap() && len(jsonresp.QUICHandshake) > 0:
				// all good: we have QUIC enabled and we get an HTTP/3 response

			case (tc.enableQUIC.IsNone() || tc.enableQUIC.Unwrap() == false) && len(jsonresp.QUICHandshake) <= 0:
				// all good: either default behavior or QUIC not enabled and not HTTP/3 response

			default:
				t.Fatalf(
					"tc.enableQUIC.IsNone() = %v, tc.enableQUIC.UnwrapOr(false) = %v, jsonresp.QUICHandshake = %v",
					tc.enableQUIC.IsNone(),
					tc.enableQUIC.UnwrapOr(false),
					jsonresp.QUICHandshake,
				)
			}
		})
	}
}
