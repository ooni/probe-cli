// Package riseupvpn contains the RiseupVPN network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-026-riseupvpn.md
package riseupvpn

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName      = "riseupvpn"
	testVersion   = "0.3.0"
	eipServiceURL = "https://api.black.riseup.net:443/3/config/eip-service.json"
	providerURL   = "https://riseup.net/provider.json"
	geoServiceURL = "https://api.black.riseup.net:9001/json"
	tcpConnect    = "tcpconnect://"
)

// EIPServiceV3 is the main JSON object returned by eip-service.json.
type EIPServiceV3 struct {
	Gateways []GatewayV3
}

// CapabilitiesV3 is a list of transports a gateway supports
type CapabilitiesV3 struct {
	Transport []TransportV3
}

// GatewayV3 describes a gateway.
type GatewayV3 struct {
	Capabilities CapabilitiesV3
	Host         string
	IPAddress    string `json:"ip_address"`
}

// TransportV3 describes a transport.
type TransportV3 struct {
	Type      string
	Protocols []string
	Ports     []string
	Options   map[string]string
}

// GatewayConnection describes the connection to a riseupvpn gateway.
type GatewayConnection struct {
	IP            string `json:"ip"`
	Port          int    `json:"port"`
	TransportType string `json:"transport_type"`
}

// Config contains the riseupvpn experiment config.
type Config struct {
	urlgetter.Config
}

// TestKeys contains riseupvpn test keys.
type TestKeys struct {
	urlgetter.TestKeys
	APIFailures  []string `json:"api_failures"`
	CACertStatus bool     `json:"ca_cert_status"`
}

// NewTestKeys creates new riseupvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		APIFailures:  []string{},
		CACertStatus: true,
	}
}

// UpdateProviderAPITestKeys updates the TestKeys using the given MultiOutput result.
func (tk *TestKeys) UpdateProviderAPITestKeys(v urlgetter.MultiOutput) {
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, v.TestKeys.Queries...)
	tk.Requests = append(tk.Requests, v.TestKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, v.TestKeys.TLSHandshakes...)
	if v.TestKeys.Failure != nil {
		tk.APIFailures = append(tk.APIFailures, *v.TestKeys.Failure)
		return
	}
}

// AddGatewayConnectTestKeys updates the TestKeys using the given MultiOutput
// result of gateway connectivity testing. Sets TransportStatus to "ok" if
// any successful TCP connection could be made
func (tk *TestKeys) AddGatewayConnectTestKeys(v urlgetter.MultiOutput, transportType string) {
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
}

// AddCACertFetchTestKeys adds generic urlgetter.Get() testKeys to riseupvpn specific test keys
func (tk *TestKeys) AddCACertFetchTestKeys(testKeys urlgetter.TestKeys) {
	tk.NetworkEvents = append(tk.NetworkEvents, testKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, testKeys.Queries...)
	tk.Requests = append(tk.Requests, testKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, testKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, testKeys.TLSHandshakes...)
}

// Measurer performs the measurement.
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config

	// Getter is an optional getter to be used for testing.
	Getter urlgetter.MultiGetter
}

// ExperimentName implements ExperimentMeasurer.ExperimentName.
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion.
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements ExperimentMeasurer.Run.
func (m Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session

	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	testkeys := NewTestKeys()
	measurement.TestKeys = testkeys
	urlgetter.RegisterExtensions(measurement)

	certPool := netxlite.NewMozillaCertPool()

	// used multiple times below
	multi := urlgetter.Multi{
		Begin:   measurement.MeasurementStartTimeSaved,
		Getter:  m.Getter,
		Session: sess,
	}

	// See if we can get the certificate first
	caTarget := "https://black.riseup.net/ca.crt"
	inputs := []urlgetter.MultiInput{{
		Target: caTarget,
		Config: urlgetter.Config{
			Method:          "GET",
			FailOnHTTPError: true,
		}},
	}

	// Q: why returning early if we cannot fetch the CA or the config? Cannot we just
	// disable certificate verification and fetch the config?
	//
	// A: I do not feel comfortable with fetching without verying the certificates since
	// this means the experiment could be person-in-the-middled and forced to perform TCP
	// connect to arbitrary hosts, which maybe is harmless but still a bummer.
	//
	// TODO(https://github.com/ooni/probe/issues/2559): solve this problem by serving the
	// correct CA and the endpoints to probes using check-in v2 (aka richer input).

	nullCallbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	for entry := range multi.CollectOverall(ctx, inputs, 0, 20, "riseupvpn", nullCallbacks) {
		tk := entry.TestKeys
		testkeys.AddCACertFetchTestKeys(tk)
		if tk.Failure != nil {
			testkeys.CACertStatus = false
			testkeys.APIFailures = append(testkeys.APIFailures, *tk.Failure)
			return nil
		}
		if ok := certPool.AppendCertsFromPEM([]byte(tk.HTTPResponseBody)); !ok {
			testkeys.CACertStatus = false
			testkeys.APIFailures = append(testkeys.APIFailures, "invalid_ca")
			return nil
		}
	}

	// Now test the service endpoints using the above-fetched CA
	inputs = []urlgetter.MultiInput{
		// Here we need to provide the method explicitly. See
		// https://github.com/ooni/probe-engine/issues/827.
		{Target: providerURL, Config: urlgetter.Config{
			CertPool:        certPool,
			Method:          "GET",
			FailOnHTTPError: true,
		}},
		{Target: eipServiceURL, Config: urlgetter.Config{
			CertPool:        certPool,
			Method:          "GET",
			FailOnHTTPError: true,
		}},
		{Target: geoServiceURL, Config: urlgetter.Config{
			CertPool:        certPool,
			Method:          "GET",
			FailOnHTTPError: true,
		}},
	}
	for entry := range multi.CollectOverall(ctx, inputs, 1, 20, "riseupvpn", nullCallbacks) {
		testkeys.UpdateProviderAPITestKeys(entry)
		tk := entry.TestKeys
		if tk.Failure != nil {
			return nil
		}
	}

	// test gateways now
	gateways := parseGateways(testkeys)
	openvpnEndpoints := generateMultiInputs(gateways, "openvpn")
	obfs4Endpoints := generateMultiInputs(gateways, "obfs4")
	overallCount := 1 + len(inputs) + len(openvpnEndpoints) + len(obfs4Endpoints)
	startCount := 1 + len(inputs)

	// measure openvpn in parallel
	for entry := range multi.CollectOverall(
		ctx, openvpnEndpoints, startCount, overallCount, "riseupvpn", callbacks) {
		testkeys.AddGatewayConnectTestKeys(entry, "openvpn")
	}

	// measure obfs4 in parallel
	// TODO(bassosimone): when urlgetter is able to do obfs4 handshakes, here
	// can possibly also test for the obfs4 handshake.
	// See https://github.com/ooni/probe/issues/1463.
	startCount += len(openvpnEndpoints)
	for entry := range multi.CollectOverall(
		ctx, obfs4Endpoints, startCount, overallCount, "riseupvpn", callbacks) {
		testkeys.AddGatewayConnectTestKeys(entry, "obfs4")
	}
	return nil
}

func generateMultiInputs(gateways []GatewayV3, transportType string) []urlgetter.MultiInput {
	var gatewayInputs []urlgetter.MultiInput
	for _, gateway := range gateways {
		for _, transport := range gateway.Capabilities.Transport {
			if transport.Type != transportType {
				continue
			}
			supportsTCP := false
			for _, protocol := range transport.Protocols {
				if protocol == "tcp" {
					supportsTCP = true
				}
			}
			if !supportsTCP {
				continue
			}
			for _, port := range transport.Ports {
				tcpConnection := tcpConnect + gateway.IPAddress + ":" + port
				gatewayInputs = append(gatewayInputs, urlgetter.MultiInput{Target: tcpConnection})
			}
		}
	}
	return gatewayInputs
}

func parseGateways(testKeys *TestKeys) []GatewayV3 {
	for _, requestEntry := range testKeys.Requests {
		if requestEntry.Request.URL == eipServiceURL && requestEntry.Failure == nil {
			// TODO(bassosimone,cyberta): is it reasonable that we discard
			// the error when the JSON we fetched cannot be parsed?
			// See https://github.com/ooni/probe/issues/1432
			eipService, err := DecodeEIPServiceV3(string(requestEntry.Response.Body))
			if err == nil {
				return eipService.Gateways
			}
		}
	}
	return nil
}

// DecodeEIPServiceV3 decodes eip-service.json version 3
func DecodeEIPServiceV3(body string) (*EIPServiceV3, error) {
	var eip EIPServiceV3
	err := json.Unmarshal([]byte(body), &eip)
	if err != nil {
		return nil, err
	}
	return &eip, nil
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return Measurer{Config: config}
}

// SummaryKeys contains summary keys for this experiment.
//
// Note that this structure is part of the ABI contract with ooniprobe
// therefore we should be careful when changing it.
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	return sk, nil
}
