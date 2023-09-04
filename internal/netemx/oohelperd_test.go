package netemx

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestOOHelperDHandler(t *testing.T) {
	env := MustNewScenario(InternetScenario)
	defer env.Close()

	env.Do(func() {
		thReq := &model.THRequest{
			HTTPRequest: "https://www.example.com/",
			HTTPRequestHeaders: map[string][]string{
				"accept":          {model.HTTPHeaderAccept},
				"accept-language": {model.HTTPHeaderAcceptLanguage},
				"user-agent":      {model.HTTPHeaderUserAgent},
			},
			TCPConnect:   []string{InternetScenarioAddressWwwExampleCom},
			XQUICEnabled: true,
		}
		thReqRaw := runtimex.Try1(json.Marshal(thReq))

		//log.SetLevel(log.DebugLevel)

		httpClient := netxlite.NewHTTPClientStdlib(log.Log)

		req, err := http.NewRequest(http.MethodPost, "https://0.th.ooni.org/", bytes.NewReader(thReqRaw))
		if err != nil {
			t.Fatal(err)
		}

		resp, err := httpClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code", resp.StatusCode)
		}
		body, err := netxlite.ReadAllContext(context.Background(), resp.Body)
		if err != nil {
			t.Fatal(err)
		}

		//t.Log(string(body))

		var thResp model.THResponse
		if err := json.Unmarshal(body, &thResp); err != nil {
			t.Fatal(err)
		}

		expectedTHResp := &model.THResponse{
			TCPConnect: map[string]model.THTCPConnectResult{
				"93.184.216.34:443": {
					Status:  true,
					Failure: nil,
				},
			},
			TLSHandshake: map[string]model.THTLSHandshakeResult{
				"93.184.216.34:443": {
					ServerName: "www.example.com",
					Status:     true,
					Failure:    nil,
				},
			},
			QUICHandshake: map[string]model.THTLSHandshakeResult{
				"93.184.216.34:443": {
					ServerName: "www.example.com",
					Status:     true,
					Failure:    nil,
				},
			},
			HTTPRequest: model.THHTTPRequestResult{
				BodyLength:           194,
				DiscoveredH3Endpoint: "www.example.com:443",
				Failure:              nil,
				Title:                "Default Web Page",
				Headers: map[string]string{
					"Alt-Svc":        `h3=":443"`,
					"Content-Length": "194",
					"Content-Type":   "text/html; charset=utf-8",
					"Date":           "Thu, 24 Aug 2023 14:35:29 GMT",
				},
				StatusCode: 200,
			},
			HTTP3Request: &model.THHTTPRequestResult{
				BodyLength:           194,
				DiscoveredH3Endpoint: "",
				Failure:              nil,
				Title:                "Default Web Page",
				Headers: map[string]string{
					"Alt-Svc":      `h3=":443"`,
					"Content-Type": "text/html; charset=utf-8",
					"Date":         "Thu, 24 Aug 2023 14:35:29 GMT",
				},
				StatusCode: 200,
			},
			DNS: model.THDNSResult{
				Failure: nil,
				Addrs:   []string{"93.184.216.34"},
				ASNs:    nil,
			},
			IPInfo: map[string]*model.THIPInfo{
				"93.184.216.34": {
					ASN:   15133,
					Flags: 10,
				},
			},
		}

		if diff := cmp.Diff(expectedTHResp, &thResp); diff != "" {
			t.Fatal(diff)
		}
	})
}
