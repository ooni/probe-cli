// Package riseupvpn contains the RiseupVPN network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-026-riseupvpn.md
package riseupvpn

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName      = "riseupvpn"
	testVersion   = "0.2.0"
	eipServiceURL = "https://api.black.riseup.net:443/3/config/eip-service.json"
	providerURL   = "https://riseup.net/provider.json"
	geoServiceURL = "https://api.black.riseup.net:9001/json"
	tcpConnect    = "tcpconnect://"
)

// EipService is the main JSON object of eip-service.json.
type EipService struct {
	Gateways []GatewayV3
}

// GatewayV3 describes a gateway.
type GatewayV3 struct {
	Capabilities struct {
		Transport []TransportV3
	}
	Host      string
	IPAddress string `json:"ip_address"`
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
	APIFailure      *string             `json:"api_failure"`
	APIStatus       string              `json:"api_status"`
	CACertStatus    bool                `json:"ca_cert_status"`
	FailingGateways []GatewayConnection `json:"failing_gateways"`
	TransportStatus map[string]string   `json:"transport_status"`
}

// NewTestKeys creates new riseupvpn TestKeys.
func NewTestKeys() *TestKeys {
	return &TestKeys{
		APIFailure:      nil,
		APIStatus:       "ok",
		CACertStatus:    true,
		FailingGateways: nil,
		TransportStatus: nil,
	}
}

// UpdateProviderAPITestKeys updates the TestKeys using the given MultiOutput result.
func (tk *TestKeys) UpdateProviderAPITestKeys(v urlgetter.MultiOutput) {
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, v.TestKeys.Queries...)
	tk.Requests = append(tk.Requests, v.TestKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, v.TestKeys.TLSHandshakes...)
	if tk.APIStatus != "ok" {
		return // we already flipped the state
	}
	if v.TestKeys.Failure != nil {
		tk.APIStatus = "blocked"
		tk.APIFailure = v.TestKeys.Failure
		return
	}
}

// AddGatewayConnectTestKeys updates the TestKeys using the given MultiOutput
// result of gateway connectivity testing. Sets TransportStatus to "ok" if
// any successful TCP connection could be made
func (tk *TestKeys) AddGatewayConnectTestKeys(v urlgetter.MultiOutput, transportType string) {
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	for _, tcpConnect := range v.TestKeys.TCPConnect {
		if !tcpConnect.Status.Success {
			gatewayConnection := newGatewayConnection(tcpConnect, transportType)
			tk.FailingGateways = append(tk.FailingGateways, *gatewayConnection)
		}
	}
}

func (tk *TestKeys) updateTransportStatus(openvpnGatewayCount, obfs4GatewayCount int) {
	failingOpenvpnGateways, failingObfs4Gateways := 0, 0
	for _, gw := range tk.FailingGateways {
		if gw.TransportType == "openvpn" {
			failingOpenvpnGateways++
		} else if gw.TransportType == "obfs4" {
			failingObfs4Gateways++
		}
	}
	if failingOpenvpnGateways < openvpnGatewayCount {
		tk.TransportStatus["openvpn"] = "ok"
	} else {
		tk.TransportStatus["openvpn"] = "blocked"
	}
	if failingObfs4Gateways < obfs4GatewayCount {
		tk.TransportStatus["obfs4"] = "ok"
	} else {
		tk.TransportStatus["obfs4"] = "blocked"
	}
}

func newGatewayConnection(
	tcpConnect archival.TCPConnectEntry, transportType string) *GatewayConnection {
	return &GatewayConnection{
		IP:            tcpConnect.IP,
		Port:          tcpConnect.Port,
		TransportType: transportType,
	}
}

// AddCACertFetchTestKeys adds generic urlgetter.Get() testKeys to riseupvpn specific test keys
func (tk *TestKeys) AddCACertFetchTestKeys(testKeys urlgetter.TestKeys) {
	tk.NetworkEvents = append(tk.NetworkEvents, testKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, testKeys.Queries...)
	tk.Requests = append(tk.Requests, testKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, testKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, testKeys.TLSHandshakes...)
	if testKeys.Failure != nil {
		tk.APIStatus = "blocked"
		tk.APIFailure = tk.Failure
		tk.CACertStatus = false
	}
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
func (m Measurer) Run(ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks) error {
	ctx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	testkeys := NewTestKeys()
	measurement.TestKeys = testkeys
	urlgetter.RegisterExtensions(measurement)

	certPool := netxlite.NewDefaultCertPool()

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
	for entry := range multi.CollectOverall(ctx, inputs, 0, 50, "riseupvpn", callbacks) {
		tk := entry.TestKeys
		testkeys.AddCACertFetchTestKeys(tk)
		if tk.Failure != nil {
			// TODO(bassosimone,cyberta): should we update the testkeys
			// in this case (e.g., APIFailure?)
			// See https://github.com/ooni/probe/issues/1432.
			return nil
		}
		if ok := certPool.AppendCertsFromPEM([]byte(tk.HTTPResponseBody)); !ok {
			testkeys.CACertStatus = false
			testkeys.APIStatus = "blocked"
			errorValue := "invalid_ca"
			testkeys.APIFailure = &errorValue
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
	for entry := range multi.CollectOverall(ctx, inputs, 1, 50, "riseupvpn", callbacks) {
		testkeys.UpdateProviderAPITestKeys(entry)
	}

	// test gateways now
	testkeys.TransportStatus = map[string]string{}
	gateways := parseGateways(testkeys)
	openvpnEndpoints := generateMultiInputs(gateways, "openvpn")
	obfs4Endpoints := generateMultiInputs(gateways, "obfs4")
	overallCount := 1 + len(inputs) + len(openvpnEndpoints) + len(obfs4Endpoints)

	// measure openvpn in parallel
	for entry := range multi.CollectOverall(
		ctx, openvpnEndpoints, 1+len(inputs), overallCount, "riseupvpn", callbacks) {
		testkeys.AddGatewayConnectTestKeys(entry, "openvpn")
	}

	// measure obfs4 in parallel
	// TODO(bassosimone): when urlgetter is able to do obfs4 handshakes, here
	// can possibly also test for the obfs4 handshake.
	// See https://github.com/ooni/probe/issues/1463.
	for entry := range multi.CollectOverall(
		ctx, obfs4Endpoints, 1+len(inputs)+len(openvpnEndpoints), overallCount, "riseupvpn", callbacks) {
		testkeys.AddGatewayConnectTestKeys(entry, "obfs4")
	}

	// set transport status based on gateway test results
	testkeys.updateTransportStatus(len(openvpnEndpoints), len(obfs4Endpoints))
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
			eipService, err := DecodeEIP3(requestEntry.Response.Body.Value)
			if err == nil {
				return eipService.Gateways
			}
		}
	}
	return nil
}

// DecodeEIP3 decodes eip-service.json version 3
func DecodeEIP3(body string) (*EipService, error) {
	var eip EipService
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
// Note that this structure is part of the ABI contract with probe-cli
// therefore we should be careful when changing it.
type SummaryKeys struct {
	APIBlocked      bool              `json:"api_blocked"`
	ValidCACert     bool              `json:"valid_ca_cert"`
	FailingGateways int               `json:"failing_gateways"`
	TransportStatus map[string]string `json:"transport_status"`
	IsAnomaly       bool              `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	sk.APIBlocked = tk.APIStatus != "ok"
	sk.ValidCACert = tk.CACertStatus
	sk.FailingGateways = len(tk.FailingGateways)
	sk.TransportStatus = tk.TransportStatus
	// Note: the order in the following OR chains matter: TransportStatus
	// is nil if APIBlocked or !CACertStatus
	sk.IsAnomaly = (sk.APIBlocked || !tk.CACertStatus ||
		tk.TransportStatus["openvpn"] == "blocked" ||
		tk.TransportStatus["obfs4"] == "blocked")
	return sk, nil
}
