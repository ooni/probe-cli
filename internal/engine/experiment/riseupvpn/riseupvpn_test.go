package riseupvpn_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/riseupvpn"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/selfcensor"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	if measurer.ExperimentName() != "riseupvpn" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestGood(t *testing.T) {
	t.Skip("broken test; see https://github.com/ooni/probe/issues/1338")
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
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
	if len(tk.NetworkEvents) <= 0 {
		t.Fatal("no NetworkEvents?!")
	}
	if len(tk.Queries) <= 0 {
		t.Fatal("no Queries?!")
	}
	if len(tk.Requests) <= 0 {
		t.Fatal("no Requests?!")
	}
	if len(tk.TCPConnect) <= 0 {
		t.Fatal("no TCPConnect?!")
	}
	if len(tk.TLSHandshakes) <= 0 {
		t.Fatal("no TLSHandshakes?!")
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
				s := errorx.HTTPRoundTripOperation
				return &s
			})(),
			Failure: (func() *string {
				s := errorx.FailureEOFError
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
	if *tk.APIFailure != errorx.FailureEOFError {
		t.Fatal("invalid ApiFailure")
	}
}

func TestFailureCaCertFetch(t *testing.T) {
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	// we're cancelling immediately so that the CA Cert fetch fails
	cancel()

	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
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
}

func TestFailureEipServiceBlocked(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	selfcensor.Enable(`{"PoisonSystemDNS":{"api.black.riseup.net":["NXDOMAIN"]}}`)

	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus ")
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
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	selfcensor.Enable(`{"BlockedEndpoints":{"198.252.153.70:443":"REJECT"}}`)

	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
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
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	selfcensor.Enable(`{"BlockedEndpoints":{"198.252.153.107:9001":"REJECT"}}`)

	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
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

func TestFailureGateway(t *testing.T) {
	t.Skip("broken test; see https://github.com/ooni/probe/issues/1338")
	var testCases = [...]string{"openvpn", "obfs4"}
	eipService, err := fetchEipService()
	if err != nil {
		t.Log("Preconditions for the test are not met. Skipping due to: " + err.Error())
		t.SkipNow()
	}
	for _, tc := range testCases {
		t.Run(fmt.Sprintf("testing censored transport %s", tc), func(t *testing.T) {
			censoredGateway, err := selfCensorRandomGateway(eipService, tc)
			if err == nil {
				censorString := `{"BlockedEndpoints":{"` + censoredGateway.IP + `:` + censoredGateway.Port + `":"REJECT"}}`
				selfcensor.Enable(censorString)
			} else {
				t.Log("Preconditions for the test are not met. Skipping due to: " + err.Error())
				t.SkipNow()
			}

			// - run measurement
			runGatewayTest(t, censoredGateway)
		})
	}
}

type SelfCensoredGateway struct {
	IP   string
	Port string
}

func fetchEipService() (*riseupvpn.EipService, error) {
	// - fetch client cert and add to certpool
	caFetchClient := &http.Client{
		Timeout: time.Second * 30,
	}

	caCertResponse, err := caFetchClient.Get("https://black.riseup.net/ca.crt")
	if err != nil {
		return nil, err
	}

	var bodyString string

	if caCertResponse.StatusCode != http.StatusOK {
		return nil, errors.New("unexpected HTTP response code")
	}
	bodyBytes, err := ioutil.ReadAll(caCertResponse.Body)
	defer caCertResponse.Body.Close()

	if err != nil {
		return nil, err
	}
	bodyString = string(bodyBytes)

	certs := x509.NewCertPool()
	certs.AppendCertsFromPEM([]byte(bodyString))

	// - fetch and parse eip-service.json
	client := &http.Client{
		Timeout: time.Second * 30,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs: certs,
			},
		},
	}

	eipResponse, err := client.Get("https://api.black.riseup.net/3/config/eip-service.json")
	if err != nil {
		return nil, err
	}
	if eipResponse.StatusCode != http.StatusOK {
		return nil, errors.New("Unexpected HTTP response code")
	}

	bodyBytes, err = ioutil.ReadAll(eipResponse.Body)
	defer eipResponse.Body.Close()
	if err != nil {
		return nil, err
	}
	bodyString = string(bodyBytes)

	eipService, err := riseupvpn.DecodeEIP3(bodyString)
	if err != nil {
		return nil, err
	}
	return eipService, nil
}

func selfCensorRandomGateway(eipService *riseupvpn.EipService, transportType string) (*SelfCensoredGateway, error) {

	// - self censor random gateway
	gateways := eipService.Gateways
	if gateways == nil || len(gateways) == 0 {
		return nil, errors.New("No gateways found")
	}

	var selfcensoredGateways []SelfCensoredGateway
	for _, gateway := range gateways {
		for _, transport := range gateway.Capabilities.Transport {
			if transport.Type == transportType {
				selfcensoredGateways = append(selfcensoredGateways, SelfCensoredGateway{IP: gateway.IPAddress, Port: transport.Ports[0]})
			}
		}
	}

	if len(selfcensoredGateways) == 0 {
		return nil, errors.New("transport " + transportType + " doesn't seem to be supported.")
	}

	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	min := 0
	max := len(selfcensoredGateways) - 1
	randomIndex := rnd.Intn(max-min+1) + min
	return &selfcensoredGateways[randomIndex], nil

}

func runGatewayTest(t *testing.T, censoredGateway *SelfCensoredGateway) {
	measurer := riseupvpn.NewExperimentMeasurer(riseupvpn.Config{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sess := &mockable.Session{MockableLogger: log.Log}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(log.Log)
	err := measurer.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := measurement.TestKeys.(*riseupvpn.TestKeys)
	if tk.CACertStatus != true {
		t.Fatal("invalid CACertStatus ")
	}

	if tk.FailingGateways == nil || len(tk.FailingGateways) != 1 {
		t.Fatal("unexpected amount of failing gateways")
	}

	entry := tk.FailingGateways[0]
	if entry.IP != censoredGateway.IP || fmt.Sprint(entry.Port) != censoredGateway.Port {
		t.Fatal("unexpected failed gateway configuration")
	}

	if tk.APIStatus == "blocked" {
		t.Fatal("invalid ApiStatus")
	}

	if tk.APIFailure != nil {
		t.Fatal("ApiFailure should be null")
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
		},
		sk: riseupvpn.SummaryKeys{
			APIBlocked:  true,
			ValidCACert: true,
			IsAnomaly:   true,
		},
	}, {
		tk: riseupvpn.TestKeys{
			APIStatus:       "ok",
			CACertStatus:    false,
			FailingGateways: nil,
		},
		sk: riseupvpn.SummaryKeys{
			ValidCACert: false,
			IsAnomaly:   true,
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
		},
		sk: riseupvpn.SummaryKeys{
			FailingGateways: 1,
			IsAnomaly:       true,
			ValidCACert:     true,
		},
	}}
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
