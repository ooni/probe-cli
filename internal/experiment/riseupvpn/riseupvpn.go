// Package riseupvpn contains the RiseupVPN network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-026-riseupvpn.md
package riseupvpn

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

const (
	testName      = "riseupvpn"
	testVersion   = "0.3.0"
	eipServiceURL = "https://api.black.riseup.net:443/3/config/eip-service.json"
	providerURL   = "https://riseup.net/provider.json"
	geoServiceURL = "https://api.black.riseup.net:9001/json"
	tcpConnect    = "tcpconnect://"
)

// EipService is the main JSON object of eip-service.json.
type EipService struct {
	Gateways []GatewayV3
}

// Capabilities is a list of transports a gateway supports
type Capabilities struct {
	Transport []TransportV3
}

// GatewayV3 describes a gateway.
type GatewayV3 struct {
	Capabilities Capabilities
	Host         string
	IPAddress    string `json:"ip_address"`
	Location     string `json:"location"`
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

// GatewayLoad describes the load of a single Gateway.
type GatewayLoad struct {
	Host     string  `json:"host"`
	Fullness float64 `json:"fullness"`
	Overload bool    `json:"overload"`
}

// GeoService represents the geoService API (also known as menshen) json response
type GeoService struct {
	IPAddress      string        `json:"ip"`
	Country        string        `json:"cc"`
	City           string        `json:"city"`
	Latitude       float64       `json:"lat"`
	Longitude      float64       `json:"lon"`
	Gateways       []string      `json:"gateways"`
	SortedGateways []GatewayLoad `json:"sortedGateways"`
}

// Config contains the riseupvpn experiment config.
type Config struct {
	urlgetter.Config
}

// TestKeys contains riseupvpn test keys.
type TestKeys struct {
	urlgetter.TestKeys
	APIFailure      []string            `json:"api_failure"`
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
	if v.TestKeys.Failure != nil {
		for _, request := range v.TestKeys.Requests {
			if request.Request.URL == eipServiceURL && request.Failure != nil {
				tk.APIStatus = "blocked"
			}
		}
		tk.APIFailure = append(tk.APIFailure, *v.TestKeys.Failure)
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
	tcpConnect tracex.TCPConnectEntry, transportType string) *GatewayConnection {
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
	for entry := range multi.CollectOverall(ctx, inputs, 0, 20, "riseupvpn", callbacks) {
		tk := entry.TestKeys
		testkeys.AddCACertFetchTestKeys(tk)
		if tk.Failure != nil {
			testkeys.CACertStatus = false
			testkeys.APIFailure = append(testkeys.APIFailure, *tk.Failure)
			certPool = nil
		} else if ok := certPool.AppendCertsFromPEM([]byte(tk.HTTPResponseBody)); !ok {
			testkeys.CACertStatus = false
			testkeys.APIFailure = append(testkeys.APIFailure, "invalid_ca")
			certPool = nil
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
			NoTLSVerify:     !testkeys.CACertStatus,
		}},
		{Target: eipServiceURL, Config: urlgetter.Config{
			CertPool:        certPool,
			Method:          "GET",
			FailOnHTTPError: true,
			NoTLSVerify:     !testkeys.CACertStatus,
		}},
		{Target: geoServiceURL, Config: urlgetter.Config{
			CertPool:        certPool,
			Method:          "GET",
			FailOnHTTPError: true,
			NoTLSVerify:     !testkeys.CACertStatus,
		}},
	}

	for entry := range multi.CollectOverall(ctx, inputs, 1, 20, "riseupvpn", callbacks) {
		testkeys.UpdateProviderAPITestKeys(entry)
	}

	if testkeys.APIStatus == "blocked" {
		for _, input := range inputs {
			input.Config.Tunnel = "torsf"
		}
		for entry := range multi.CollectOverall(ctx, inputs, 1, 20, "riseupvpn", callbacks) {
			testkeys.UpdateProviderAPITestKeys(entry)
		}
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
	var eipService *EipService = nil
	var geoService *GeoService = nil
	for _, requestEntry := range testKeys.Requests {
		if requestEntry.Request.URL == eipServiceURL && requestEntry.Failure == nil {
			var err error = nil
			eipService, err = DecodeEIP3(requestEntry.Response.Body.Value)
			if err != nil {
				testKeys.APIFailure = append(testKeys.APIFailure, "invalid_eipservice_response")
				return nil
			}
		} else if requestEntry.Request.URL == geoServiceURL && requestEntry.Failure == nil {
			var err error = nil
			geoService, err = DecodeGeoService(requestEntry.Response.Body.Value)
			if err != nil {
				testKeys.APIFailure = append(testKeys.APIFailure, "invalid_geoservice_response")
			}
		}
	}
	return filterGateways(eipService, geoService)
}

// filterGateways selects a subset of available gateways supporting obfs4
func filterGateways(eipService *EipService, geoService *GeoService) []GatewayV3 {
	var result []GatewayV3 = nil
	if eipService != nil {
		locations := getLocationsUnderTest(eipService, geoService)
		for _, gateway := range eipService.Gateways {
			if !gateway.hasTransport("obfs4") ||
				!gateway.isLocationUnderTest(locations) ||
				geoService != nil && !geoService.isHealthyGateway(gateway) {
				continue
			}
			result = append(result, gateway)
			if len(result) == 3 {
				return result
			}
		}
	}
	return result
}

// getLocationsUnderTest parses all gateways supporting obfs4 and returns the two locations having most obfs4 bridges
func getLocationsUnderTest(eipService *EipService, geoService *GeoService) []string {
	var result []string = nil
	if eipService != nil {
		locationMap := map[string]int{}
		locations := []string{}
		for _, gateway := range eipService.Gateways {
			if !gateway.hasTransport("obfs4") {
				continue
			}
			if _, ok := locationMap[gateway.Location]; !ok {
				locations = append(locations, gateway.Location)
			}
			locationMap[gateway.Location] += 1
		}

		location1 := ""
		location2 := ""
		for _, location := range locations {
			if locationMap[location] > locationMap[location1] {
				location2 = location1
				location1 = location
			} else if locationMap[location] > locationMap[location2] {
				location2 = location
			}
		}
		if location1 != "" {
			result = append(result, location1)
		}
		if location2 != "" {
			result = append(result, location2)
		}
	}

	return result
}

func (gateway *GatewayV3) hasTransport(s string) bool {
	for _, transport := range gateway.Capabilities.Transport {
		if s == transport.Type {
			return true
		}
	}
	return false
}

func (gateway *GatewayV3) isLocationUnderTest(locations []string) bool {
	for _, location := range locations {
		if location == gateway.Location {
			return true
		}
	}
	return false
}

func (geoService *GeoService) isHealthyGateway(gateway GatewayV3) bool {
	if geoService.SortedGateways == nil {
		// Earlier versions of the geoservice don't include the sorted gateway list containing the load info,
		// so we can't say anything about the load of a gateway in that case.
		// We assume it's an healthy location. Riseup will switch to the updated API soon *fingers crossed*
		return true
	}
	for _, gatewayLoad := range geoService.SortedGateways {
		if gatewayLoad.Host == gateway.Host {
			return !gatewayLoad.Overload
		}
	}

	// gateways that are not included in the geoservice should be considered unusable
	return false
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

// DecodeGeoService decodes geoService  json
func DecodeGeoService(body string) (*GeoService, error) {
	var gs GeoService
	err := json.Unmarshal([]byte(body), &gs)
	if err != nil {
		return nil, err
	}
	return &gs, nil
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
