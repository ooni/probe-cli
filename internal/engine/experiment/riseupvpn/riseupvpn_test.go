package riseupvpn_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	provider = `{
		"api_uri": "https://api.black.riseup.net:443",
		"api_version": "3",
		"ca_cert_fingerprint": "SHA256: a5244308a1374709a9afce95e3ae47c1b44bc2398c0a70ccbf8b3a8a97f29494",
		"ca_cert_uri": "https://black.riseup.net/ca.crt",
		"default_language": "en",
		"description": {
		  "en": "Riseup is a non-profit collective in Seattle that provides online communication tools for people and groups working toward liberatory social change."
		},
		"domain": "riseup.net",
		"enrollment_policy": "closed",
		"languages": [
		  "en"
		],
		"name": {
		  "en": "Riseup Networks"
		},
		"service": {
		  "allow_anonymous": true,
		  "allow_free": true,
		  "allow_limited_bandwidth": false,
		  "allow_paid": false,
		  "allow_registration": false,
		  "allow_unlimited_bandwidth": true,
		  "bandwidth_limit": 102400,
		  "default_service_level": 1,
		  "levels": {
			"1": {
			  "description": "Please donate.",
			  "name": "free"
			}
		  }
		},
		"services": [
		  "openvpn"
		]
	  }`
	eipservice = `{
		"gateways": [
		  {
			"capabilities": {
			  "adblock": false,
			  "filter_dns": false,
			  "limited": false,
			  "transport":[
				{
				  "type":"openvpn",
				  "protocols":[
					"tcp"
				  ],
				  "ports":[
					"443"
				  ]
				}
			  ],
			  "user_ips": false
			},
			"host": "test1.riseup.net",
			"ip_address": "123.456.123.456",
			"location": "paris"
		  },
		  {
			"capabilities": {
			  "adblock": false,
			  "filter_dns": false,
			  "limited": false,
			  "transport":[
				{
				  "type":"obfs4",
				  "protocols":[
					"tcp"
				  ],
				  "ports":[
					"23042"
				  ],
				  "options": {
					"cert": "XXXXXXXXXXXXXXXXXXXXXXXXX",
					"iatMode": "0"
				  }
				},              
				{
				  "type":"openvpn",
				  "protocols":[
					"tcp"
				  ],
				  "ports":[
					"443"
				  ]
				}
			  ],
			  "user_ips": false
			},
			"host": "test2.riseup.net",
			"ip_address": "234.345.234.345",
			"location": "seattle"
		  }
		],
		"locations": {
		  "paris": {
			"country_code": "FR",
			"hemisphere": "N",
			"name": "Paris",
			"timezone": "+2"
		  },
		  "seattle": {
			"country_code": "US",
			"hemisphere": "N",
			"name": "Seattle",
			"timezone": "-7"
		  }
		},
		"openvpn_configuration": {
		  "auth": "SHA1",
		  "cipher": "AES-128-CBC",
		  "keepalive": "10 30",
		  "tls-cipher": "DHE-RSA-AES128-SHA",
		  "tun-ipv6": true
		},
		"serial": 3,
		"version": 3
	  }`
	geoservice = `{"ip":"51.15.0.88","cc":"NL","city":"Haarlem","lat":52.381,"lon":4.6275,"gateways":["test1.riseup.net","test2.riseup.net"]}`
	cacert     = `-----BEGIN CERTIFICATE-----
MIIFjTCCA3WgAwIBAgIBATANBgkqhkiG9w0BAQ0FADBZMRgwFgYDVQQKDA9SaXNl
dXAgTmV0d29ya3MxGzAZBgNVBAsMEmh0dHBzOi8vcmlzZXVwLm5ldDEgMB4GA1UE
AwwXUmlzZXVwIE5ldHdvcmtzIFJvb3QgQ0EwHhcNMTQwNDI4MDAwMDAwWhcNMjQw
NDI4MDAwMDAwWjBZMRgwFgYDVQQKDA9SaXNldXAgTmV0d29ya3MxGzAZBgNVBAsM
Emh0dHBzOi8vcmlzZXVwLm5ldDEgMB4GA1UEAwwXUmlzZXVwIE5ldHdvcmtzIFJv
b3QgQ0EwggIiMA0GCSqGSIb3DQEBAQUAA4ICDwAwggIKAoICAQC76J4ciMJ8Sg0m
TP7DF2DT9zNe0Csk4myoMFC57rfJeqsAlJCv1XMzBmXrw8wq/9z7XHv6n/0sWU7a
7cF2hLR33ktjwODlx7vorU39/lXLndo492ZBhXQtG1INMShyv+nlmzO6GT7ESfNE
LliFitEzwIegpMqxCIHXFuobGSCWF4N0qLHkq/SYUMoOJ96O3hmPSl1kFDRMtWXY
iw1SEKjUvpyDJpVs3NGxeLCaA7bAWhDY5s5Yb2fA1o8ICAqhowurowJpW7n5ZuLK
5VNTlNy6nZpkjt1QycYvNycffyPOFm/Q/RKDlvnorJIrihPkyniV3YY5cGgP+Qkx
HUOT0uLA6LHtzfiyaOqkXwc4b0ZcQD5Vbf6Prd20Ppt6ei0zazkUPwxld3hgyw58
m/4UIjG3PInWTNf293GngK2Bnz8Qx9e/6TueMSAn/3JBLem56E0WtmbLVjvko+LF
PM5xA+m0BmuSJtrD1MUCXMhqYTtiOvgLBlUm5zkNxALzG+cXB28k6XikXt6MRG7q
hzIPG38zwkooM55yy5i1YfcIi5NjMH6A+t4IJxxwb67MSb6UFOwg5kFokdONZcwj
shczHdG9gLKSBIvrKa03Nd3W2dF9hMbRu//STcQxOailDBQCnXXfAATj9pYzdY4k
ha8VCAREGAKTDAex9oXf1yRuktES4QIDAQABo2AwXjAdBgNVHQ4EFgQUC4tdmLVu
f9hwfK4AGliaet5KkcgwDgYDVR0PAQH/BAQDAgIEMAwGA1UdEwQFMAMBAf8wHwYD
VR0jBBgwFoAUC4tdmLVuf9hwfK4AGliaet5KkcgwDQYJKoZIhvcNAQENBQADggIB
AGzL+GRnYu99zFoy0bXJKOGCF5XUXP/3gIXPRDqQf5g7Cu/jYMID9dB3No4Zmf7v
qHjiSXiS8jx1j/6/Luk6PpFbT7QYm4QLs1f4BlfZOti2KE8r7KRDPIecUsUXW6P/
3GJAVYH/+7OjA39za9AieM7+H5BELGccGrM5wfl7JeEz8in+V2ZWDzHQO4hMkiTQ
4ZckuaL201F68YpiItBNnJ9N5nHr1MRiGyApHmLXY/wvlrOpclh95qn+lG6/2jk7
3AmihLOKYMlPwPakJg4PYczm3icFLgTpjV5sq2md9bRyAg3oPGfAuWHmKj2Ikqch
Td5CHKGxEEWbGUWEMP0s1A/JHWiCbDigc4Cfxhy56CWG4q0tYtnc2GMw8OAUO6Wf
Xu5pYKNkzKSEtT/MrNJt44tTZWbKV/Pi/N2Fx36my7TgTUj7g3xcE9eF4JV2H/sg
tsK3pwE0FEqGnT4qMFbixQmc8bGyuakr23wjMvfO7eZUxBuWYR2SkcP26sozF9PF
tGhbZHQVGZUTVPyvwahMUEhbPGVerOW0IYpxkm0x/eaWdTc4vPpf/rIlgbAjarnJ
UN9SaWRlWKSdP4haujnzCoJbM7dU9bjvlGZNyXEekgeT0W2qFeGGp+yyUWw8tNsp
0BuC1b7uW/bBn/xKm319wXVDvBgZgcktMolak39V7DVO
-----END CERTIFICATE-----`

	// TODO(bassosimone): maybe we can switch this test to internal
	// testing (since now it's all unit tested!) and just use the
	// same constants that are used in riseupvpn.go.

	eipserviceurl = "https://api.black.riseup.net:443/3/config/eip-service.json"
	providerurl   = "https://riseup.net/provider.json"
	geoserviceurl = "https://api.black.riseup.net:9001/json"
	cacerturl     = "https://black.riseup.net/ca.crt"
	openvpnurl1   = "tcpconnect://234.345.234.345:443"
	openvpnurl2   = "tcpconnect://123.456.123.456:443"
	obfs4url1     = "tcpconnect://234.345.234.345:23042"
)

var RequestResponse = map[string]string{
	eipserviceurl: eipservice,
	providerurl:   provider,
	geoserviceurl: geoservice,
	cacerturl:     cacert,
	openvpnurl1:   "",
	openvpnurl2:   "",
	obfs4url1:     "",
}

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	if measurer.ExperimentName() != "riseupvpn" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.2.0" {
		t.Fatal("unexpected version")
	}
}

func TestGood(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: true,
		providerurl:   true,
		geoserviceurl: true,
		openvpnurl1:   true,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))

	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.Agent != "" {
		t.Fatal("unexpected Agent: " + tk.Agent)
	}
	if tk.FailedOperation != nil {
		t.Fatal("unexpected FailedOperation")
	}
	if tk.Failure != nil {
		t.Fatal("unexpected Failure")
	}
	if tk.APIFailure != nil {
		t.Fatal("unexpected ApiFailure")
	}
	if tk.APIStatus != "ok" {
		t.Fatal("unexpected ApiStatus")
	}
	if tk.CACertStatus != true {
		t.Fatal("unexpected CaCertStatus")
	}
	if tk.FailingGateways != nil {
		t.Fatal("unexpected FailingGateways value")
	}
	if tk.TransportStatus == nil {
		t.Fatal("unexpected nil TransportStatus struct ")
	}
	if tk.TransportStatus["openvpn"] != "ok" {
		t.Fatal("unexpected openvpn transport status")
	}
}

// TestUpdateWithMixedResults tests if one operation failed
// ApiStatus is considered as blocked
func TestUpdateWithMixedResults(t *testing.T) {
	tk := riseupvpn.NewTestKeys()
	tk.UpdateProviderAPITestKeys(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://api.black.riseup.net:443/3/config/eip-service.json",
		},
		TestKeys: urlgetter.TestKeys{
			HTTPResponseStatus: 200,
		},
	})
	tk.UpdateProviderAPITestKeys(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://riseup.net/provider.json",
		},
		TestKeys: urlgetter.TestKeys{
			FailedOperation: (func() *string {
				s := netxlite.HTTPRoundTripOperation
				return &s
			})(),
			Failure: (func() *string {
				s := netxlite.FailureEOFError
				return &s
			})(),
		},
	})
	tk.UpdateProviderAPITestKeys(urlgetter.MultiOutput{
		Input: urlgetter.MultiInput{
			Config: urlgetter.Config{Method: "GET"},
			Target: "https://api.black.riseup.net:9001/json",
		},
		TestKeys: urlgetter.TestKeys{
			HTTPResponseStatus: 200,
		},
	})
	if tk.APIStatus != "blocked" {
		t.Fatal("ApiStatus should be blocked")
	}
	if *tk.APIFailure != netxlite.FailureEOFError {
		t.Fatal("invalid ApiFailure")
	}
	if tk.FailingGateways != nil {
		t.Fatal("invalid FailingGateways")
	}
	if tk.TransportStatus != nil {
		t.Fatal("invalid TransportStatus")
	}
}

func TestInvalidCaCert(t *testing.T) {
	requestResponseMap := map[string]string{
		eipserviceurl: eipservice,
		providerurl:   provider,
		geoserviceurl: geoservice,
		cacerturl:     "invalid",
		openvpnurl1:   "",
		openvpnurl2:   "",
		obfs4url1:     "",
	}
	measurer := riseupvpn.Measurer{
		Config: riseupvpn.Config{},
		Getter: generateMockGetter(requestResponseMap, map[string]bool{
			cacerturl:     true,
			eipserviceurl: true,
			providerurl:   true,
			geoserviceurl: true,
			openvpnurl1:   false,
			openvpnurl2:   true,
			obfs4url1:     true,
		}),
	}
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus == true {
		t.Fatal("unexpected CaCertStatus")
	}
	if tk.APIStatus != "blocked" {
		t.Fatal("ApiStatus should be blocked")
	}
	if tk.FailingGateways != nil {
		t.Fatal("invalid FailingGateways")
	}
	if tk.TransportStatus != nil {
		t.Fatal("invalid TransportStatus")
	}
}

func TestFailureCaCertFetch(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     false,
		eipserviceurl: true,
		providerurl:   true,
		geoserviceurl: true,
		openvpnurl1:   true,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))

	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != false {
		t.Fatal("invalid CACertStatus ")
	}
	if tk.APIStatus != "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure != nil {
		t.Fatal("ApiFailure should be null")
	}
	if len(tk.Requests) > 1 {
		t.Fatal("Unexpected requests")
	}
	if tk.FailingGateways != nil {
		t.Fatal("invalid FailingGateways")
	}
	if tk.TransportStatus != nil {
		t.Fatal("invalid TransportStatus")
	}
}

func TestFailureEipServiceBlocked(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: false,
		providerurl:   true,
		geoserviceurl: true,
		openvpnurl1:   true,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus")
	}

	for _, entry := range tk.Requests {
		if entry.Request.URL == "https://api.black.riseup.net:443/3/config/eip-service.json" {
			if entry.Failure == nil {
				t.Fatal("Failure for " + entry.Request.URL + " should not be null")
			}
		}
	}

	if tk.APIStatus != "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure == nil {
		t.Fatal("ApiFailure should not be null")
	}
}

func TestFailureProviderUrlBlocked(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: true,
		providerurl:   false,
		geoserviceurl: true,
		openvpnurl1:   true,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)

	for _, entry := range tk.Requests {
		if entry.Request.URL == "https://riseup.net/provider.json" {
			if entry.Failure == nil {
				t.Fatal("Failure for " + entry.Request.URL + " should not be null")
			}
		}
	}

	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus ")
	}
	if tk.APIStatus != "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure == nil {
		t.Fatal("ApiFailure should not be null")
	}
}

func TestFailureGeoIpServiceBlocked(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: true,
		providerurl:   true,
		geoserviceurl: false,
		openvpnurl1:   true,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus ")
	}

	for _, entry := range tk.Requests {
		if entry.Request.URL == "https://api.black.riseup.net:9001/json" {
			if entry.Failure == nil {
				t.Fatal("Failure for " + entry.Request.URL + " should not be null")
			}
		}
	}

	if tk.APIStatus != "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure == nil {
		t.Fatal("ApiFailure should not be null")
	}
}

func TestFailureGateway1(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: true,
		providerurl:   true,
		geoserviceurl: true,
		openvpnurl1:   false,
		openvpnurl2:   true,
		obfs4url1:     true,
	}))
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus ")
	}

	if tk.FailingGateways == nil || len(tk.FailingGateways) != 1 {
		t.Fatal("unexpected amount of failing gateways")
	}

	gw := tk.FailingGateways[0]
	if gw.IP != "234.345.234.345" {
		t.Fatal("invalid failed gateway ip: " + fmt.Sprint(gw.IP))
	}
	if gw.Port != 443 {
		t.Fatal("invalid failed gateway port: " + fmt.Sprint(gw.Port))
	}
	if gw.TransportType != "openvpn" {
		t.Fatal("invalid failed transport type: " + fmt.Sprint(gw.TransportType))
	}

	if tk.APIStatus == "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure != nil {
		t.Fatal("ApiFailure should be null")
	}

	if tk.TransportStatus == nil || tk.TransportStatus["openvpn"] == "blocked" {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}

	if tk.TransportStatus == nil || tk.TransportStatus["obfs4"] == "blocked" {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}
}

func TestFailureTransport(t *testing.T) {
	measurement := runDefaultMockTest(t, generateDefaultMockGetter(map[string]bool{
		cacerturl:     true,
		eipserviceurl: true,
		providerurl:   true,
		geoserviceurl: true,
		openvpnurl1:   false,
		openvpnurl2:   false,
		obfs4url1:     false,
	}))
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)

	if tk.TransportStatus == nil || tk.TransportStatus["openvpn"] != "blocked" {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}

	if tk.TransportStatus == nil || tk.TransportStatus["obfs4"] != "blocked" {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}
}

func TestMissingTransport(t *testing.T) {
	eipService, err := riseupvpn.DecodeEIP3(eipservice)
	if err != nil {
		t.Fatal("Preconditions for the test are not met.")
	}

	//remove obfs4 capability from 2. gateway so that our
	//mock provider supports only openvpn
	index := -1
	transports := eipService.Gateways[1].Capabilities.Transport
	for i, transport := range transports {
		if transport.Type == "obfs4" {
			index = i
			break
		}
	}
	if index == -1 {
		t.Fatal("Preconditions for the test are not met. Default eipservice string should contain obfs4 transport.")
	}

	transports[index] = transports[len(transports)-1]
	transports = transports[:len(transports)-1]
	eipService.Gateways[1].Capabilities.Transport = transports
	eipservicejson, err := json.Marshal(eipservice)
	if err != nil {
		t.Fatal(err)
	}

	requestResponseMap := map[string]string{
		eipserviceurl: string(eipservicejson),
		providerurl:   provider,
		geoserviceurl: geoservice,
		cacerturl:     cacert,
		openvpnurl1:   "",
		openvpnurl2:   "",
		obfs4url1:     "",
	}

	measurer := riseupvpn.Measurer{
		Config: riseupvpn.Config{},
		Getter: generateMockGetter(requestResponseMap, map[string]bool{
			cacerturl:     true,
			eipserviceurl: true,
			providerurl:   true,
			geoserviceurl: true,
			openvpnurl1:   true,
			openvpnurl2:   true,
			obfs4url1:     false,
		}),
	}

	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err = measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.TransportStatus == nil || tk.TransportStatus["openvpn"] != "blocked" {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}

	if _, found := tk.TransportStatus["obfs"]; found {
		t.Fatal("invalid TransportStatus: " + fmt.Sprint(tk.TransportStatus))
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &riseupvpn.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	tests := []struct {
		tk riseupvpn.TestKeys
		sk riseupvpn.SummaryKeys
	}{{
		tk: riseupvpn.TestKeys{
			APIStatus:       "blocked",
			CACertStatus:    true,
			FailingGateways: nil,
			TransportStatus: nil,
		},
		sk: riseupvpn.SummaryKeys{
			APIBlocked:      true,
			ValidCACert:     true,
			IsAnomaly:       true,
			TransportStatus: nil,
			FailingGateways: 0,
		},
	}, {
		tk: riseupvpn.TestKeys{
			APIStatus:       "ok",
			CACertStatus:    false,
			FailingGateways: nil,
			TransportStatus: nil,
		},
		sk: riseupvpn.SummaryKeys{
			ValidCACert:     false,
			IsAnomaly:       true,
			FailingGateways: 0,
			TransportStatus: nil,
		},
	}, {
		tk: riseupvpn.TestKeys{
			APIStatus:    "ok",
			CACertStatus: true,
			FailingGateways: []riseupvpn.GatewayConnection{{
				IP:            "1.1.1.1",
				Port:          443,
				TransportType: "obfs4",
			}},
			TransportStatus: map[string]string{
				"obfs4":   "blocked",
				"openvpn": "ok",
			},
		},
		sk: riseupvpn.SummaryKeys{
			FailingGateways: 1,
			IsAnomaly:       true,
			ValidCACert:     true,
			TransportStatus: map[string]string{
				"obfs4":   "blocked",
				"openvpn": "ok",
			},
		},
	}, {
		tk: riseupvpn.TestKeys{
			APIStatus:       "ok",
			CACertStatus:    true,
			FailingGateways: nil,
			TransportStatus: map[string]string{
				"openvpn": "ok",
			},
		},
		sk: riseupvpn.SummaryKeys{
			ValidCACert:     true,
			IsAnomaly:       false,
			FailingGateways: 0,
			TransportStatus: map[string]string{
				"openvpn": "ok",
			},
		},
	},
	}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &riseupvpn.Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(riseupvpn.SummaryKeys)
			if diff := cmp.Diff(tt.sk, sk); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func generateMockGetter(requestResponse map[string]string, responseStatus map[string]bool) urlgetter.MultiGetter {
	return func(ctx context.Context, g urlgetter.Getter) (urlgetter.TestKeys, error) {
		url := g.Target
		responseBody, foundRequest := requestResponse[url]
		isSuccessStatus, foundStatus := responseStatus[url]
		if !foundRequest || !foundStatus {
			return urlgetter.TestKeys{}, errors.New("request or status not found")
		}

		var failure *string
		var failedOperation *string
		isBlocked := !isSuccessStatus
		var responseStatus int64 = 200

		if isBlocked {
			responseBody = ""
			eofError := io.EOF.Error()
			failure = &eofError
			connectOperation := netxlite.ConnectOperation
			failedOperation = &connectOperation
			responseStatus = 0
		}

		tcpConnect := archival.TCPConnectEntry{
			// use some dummy IP/Port combination for URLs, we don't do DNS resolution
			IP:   "123.456.234.123",
			Port: 443,
			Status: archival.TCPConnectStatus{
				Success: isSuccessStatus,
				Blocked: &isBlocked,
				Failure: failure,
			},
		}
		if strings.Contains(url, "tcpconnect://") {
			ipPort := strings.Split(strings.Split(url, "//")[1], ":")
			port, err := strconv.ParseInt(ipPort[1], 10, 32)
			if err == nil {
				tcpConnect.IP = ipPort[0]
				tcpConnect.Port = int(port)
			}
		}

		tk := urlgetter.TestKeys{
			Failure:            failure,
			FailedOperation:    failedOperation,
			HTTPResponseStatus: responseStatus,
			HTTPResponseBody:   responseBody,
			Requests: []archival.RequestEntry{{
				Failure: failure,
				Request: archival.HTTPRequest{
					URL:             url,
					Body:            archival.MaybeBinaryValue{},
					BodyIsTruncated: false,
				},
				Response: archival.HTTPResponse{
					Body: archival.HTTPBody{
						Value: responseBody,
					},
					BodyIsTruncated: false,
				}},
			},
			TCPConnect: []archival.TCPConnectEntry{tcpConnect},
		}
		return tk, nil
	}
}

func generateDefaultMockGetter(responseStatuses map[string]bool) urlgetter.MultiGetter {
	return generateMockGetter(RequestResponse, responseStatuses)
}

func runDefaultMockTest(t *testing.T, multiGetter urlgetter.MultiGetter) *model.Measurement {
	measurer := riseupvpn.Measurer{
		Config: riseupvpn.Config{},
		Getter: multiGetter,
	}

	measurement := new(model.Measurement)
	err := measurer.Run(
		context.Background(),
		&mockable.Session{
			MockableLogger: log.Log,
		},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)

	if err != nil {
		t.Fatal(err)
	}
	return measurement
}
