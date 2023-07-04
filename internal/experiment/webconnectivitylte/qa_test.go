package webconnectivitylte_test

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/version"
)

// qaWebServerAddress is the address of www.example.com and www.example.org.
const qaWebServerAddress = "93.184.216.34"

// qaZeroTHOoniOrg is the address of 0.th.ooni.org.
const qaZeroTHOoniOrg = "104.248.30.161"

// qaNewMockedTestHelper returns an [http.Handler] that returns the expected TH response
// based on the configuration we setup in [qaNewEnvironment].
func qaNewMockedTestHelper() http.Handler {
	// TODO(bassosimone,kelmenhorst): we should use the real TH code rather than this fragile mock

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// read raw request body
		rawRequest, err := netxlite.ReadAllContext(r.Context(), r.Body)
		if err != nil {
			// it does not make sense to send a response here because the connection
			// has been closed while reading the body
			return
		}

		// parse raw request body
		var request model.THRequest
		if err := json.Unmarshal(rawRequest, &request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// parse the raw request URL
		URL, err := url.Parse(request.HTTPRequest)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// create the endpoint
		var endpoint string
		switch URL.Scheme {
		case "http":
			endpoint = net.JoinHostPort(URL.Hostname(), "80")

		case "https":
			endpoint = net.JoinHostPort(URL.Hostname(), "443")

		default:
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// create a fake response
		response := &model.THResponse{
			TCPConnect: map[string]model.THTCPConnectResult{
				endpoint: {
					Status:  true,
					Failure: nil,
				},
			},
			TLSHandshake: map[string]model.THTLSHandshakeResult{
				endpoint: {
					ServerName: URL.Hostname(),
					Status:     true,
					Failure:    nil,
				},
			},
			QUICHandshake: map[string]model.THTLSHandshakeResult{},
			HTTPRequest: model.THHTTPRequestResult{
				BodyLength:           0,
				DiscoveredH3Endpoint: "",
				Failure:              nil,
				Title:                "Default Web Page",
				Headers:              map[string]string{},
				StatusCode:           200,
			},
			HTTP3Request: nil,
			DNS: model.THDNSResult{
				Failure: nil,
				Addrs:   []string{qaWebServerAddress},
				ASNs:    []int64{15133},
			},
			IPInfo: map[string]*model.THIPInfo{
				qaWebServerAddress: {
					ASN:   15133,
					Flags: model.THIPInfoFlagResolvedByTH | model.THIPInfoFlagResolvedByProbe,
				},
			},
		}

		// serialize the response
		rawResponse, err := json.Marshal(response)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// write the response
		w.Write(rawResponse)
	})
}

// qaAddExampleDomains adds the www.example.com and www.example.org domains to the config.
func qaAddExampleDomains(config *netem.DNSConfig) {
	config.AddRecord("www.example.com", "www.example.com", qaWebServerAddress)
	config.AddRecord("www.example.org", "www.example.org", qaWebServerAddress)
}

// qaAddTHDomains adds the {0,1,2,3}.th.ooni.org domains to the config.
func qaAddTHDomains(config *netem.DNSConfig) {
	config.AddRecord("0.th.ooni.org", "0.th.ooni.org", qaZeroTHOoniOrg)
}

func qaNewEnvironment(clientDNSConfig *netem.DNSConfig) *netemx.Environment {
	// clientConfig configures the client topology
	clientConfig := &netemx.ClientConfig{
		ClientAddr:   "", // use the default
		DNSConfig:    clientDNSConfig,
		ResolverAddr: "", // use the default
	}

	// create the configuration of the uncensored DNS server.
	serversDNSConfig := netem.NewDNSConfig()
	serversDNSConfig.AddRecord("www.example.com", "www.example.com", qaWebServerAddress)
	serversDNSConfig.AddRecord("www.example.org", "www.example.org", qaWebServerAddress)

	// create the overall configuration for the servers.
	serversConfig := &netemx.ServersConfig{
		DNSConfig:    serversDNSConfig,
		ResolverAddr: "8.8.4.4", // this is what LTE uses by default
		Servers: []netemx.ConfigServerStack{{
			ServerAddr: qaWebServerAddress,
			HTTPServers: []netemx.ConfigHTTPServer{{
				Port:    80,
				QUIC:    false,
				Handler: nil, // use the default
			}, {
				Port:    443,
				QUIC:    false,
				Handler: nil, // use the default
			}, {
				Port:    443,
				QUIC:    true,
				Handler: nil, // use the default
			}},
		}, {
			ServerAddr: qaZeroTHOoniOrg,
			HTTPServers: []netemx.ConfigHTTPServer{{
				Port:    443,
				QUIC:    false,
				Handler: qaNewMockedTestHelper(),
			}},
		}},
	}

	// return the environment
	return netemx.NewEnvironment(clientConfig, serversConfig)
}

// qaNewSession creates a new mocked session.
func qaNewSession(client model.HTTPClient) model.ExperimentSession {
	return &mocks.Session{
		MockGetTestHelpersByName: func(name string) ([]model.OOAPIService, bool) {
			output := []model.OOAPIService{{
				Address: (&url.URL{Host: qaZeroTHOoniOrg, Scheme: "https", Path: "/"}).String(),
				Type:    "https",
				Front:   "",
			}}
			return output, true
		},
		MockDefaultHTTPClient: func() model.HTTPClient {
			return client
		},
		MockFetchPsiphonConfig: nil,
		MockFetchTorTargets:    nil,
		MockKeyValueStore:      nil,
		MockLogger: func() model.Logger {
			return log.Log
		},
		MockMaybeResolverIP:  nil,
		MockProbeASNString:   nil,
		MockProbeCC:          nil,
		MockProbeIP:          nil,
		MockProbeNetworkName: nil,
		MockProxyURL:         nil,
		MockResolverIP:       nil,
		MockSoftwareName:     nil,
		MockSoftwareVersion:  nil,
		MockTempDir:          nil,
		MockTorArgs:          nil,
		MockTorBinary:        nil,
		MockTunnelDir:        nil,
		MockUserAgent: func() string {
			return model.HTTPHeaderUserAgent
		},
		MockNewExperimentBuilder: nil,
		MockNewSubmitter:         nil,
		MockCheckIn:              nil,
	}
}

// qaRunWithURL runs the QA check with the given URL.
//
// Arguments:
//
// - input is the URL to measure;
//
// - setDNSClientConfig is called to set the DNS client config;
//
// - setDPI is called to configure the DPI engine.
//
// This function returns either a measurement or an error.
func qaRunWithURL(input string, setDNSClientConfig func(*netem.DNSConfig),
	setDPI func(*netem.DPIEngine)) (*model.Measurement, error) {
	// create and initialize the DNS client config.
	dnsconfig := netem.NewDNSConfig()
	setDNSClientConfig(dnsconfig)

	// create netem environment
	env := qaNewEnvironment(dnsconfig)
	defer env.Close()

	// possibly configure DPI rules
	setDPI(env.DPIEngine())

	// create the measurer and the context
	measurer := webconnectivitylte.NewExperimentMeasurer(&webconnectivitylte.Config{})
	ctx := context.Background()

	// create a new measurement
	t0 := time.Now().UTC()
	measurement := &model.Measurement{
		Annotations:               nil,
		DataFormatVersion:         "0.2.0",
		Extensions:                nil,
		ID:                        "",
		Input:                     model.MeasurementTarget(input),
		InputHashes:               nil,
		MeasurementStartTime:      t0.Format(model.MeasurementDateFormat),
		MeasurementStartTimeSaved: t0,
		Options:                   []string{},
		ProbeASN:                  "AS24960",
		ProbeCC:                   "DE",
		ProbeCity:                 "",
		ProbeIP:                   "127.0.0.1",
		ProbeNetworkName:          "Hetzner Online GmbH",
		ReportID:                  "",
		ResolverASN:               "AS24940",
		ResolverIP:                "78.46.173.81",
		ResolverNetworkName:       "Hetzner Online GmbH",
		SoftwareName:              "ooniprobe",
		SoftwareVersion:           version.Version,
		TestHelpers: map[string]any{
			"backend": map[string]string{
				"address": "https://0.th.ooni.org",
				"type":    "https",
			},
		},
		TestKeys:           nil,
		TestName:           measurer.ExperimentName(),
		MeasurementRuntime: 0,
		TestStartTime:      t0.Format(model.MeasurementDateFormat),
		TestVersion:        measurer.ExperimentVersion(),
	}

	var err error
	env.Do(func() {
		// create an HTTP client inside the env.Do function so we're using netem
		httpClient := netxlite.NewHTTPClientStdlib(log.Log)
		arguments := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: measurement,
			Session:     qaNewSession(httpClient),
		}

		// run the experiment
		err = measurer.Run(ctx, arguments)

		// compute the total measurement runtime
		runtime := time.Since(t0)
		measurement.MeasurementRuntime = runtime.Seconds()
	})

	// handle the failure case
	if err != nil {
		return nil, err
	}

	// handle the successful case
	return measurement, nil
}

// TestQACleartextWAI is a test where we fetch from a cleartext page.
func TestQACleartextWAI(t *testing.T) {
	log.SetLevel(log.DebugLevel)
	measurement, err := qaRunWithURL(
		"http://www.example.com",
		func(d *netem.DNSConfig) {
			qaAddTHDomains(d)
			qaAddExampleDomains(d)
		},
		func(d *netem.DPIEngine) {
			// nothing
		},
	)

	// fail the test on error
	if err != nil {
		t.Fatal(err)
	}

	// TODO(bassosimone,kelmenhorst): check the test keys
	_ = measurement
}

// TestQASecureWAI is a test where we fetch from a secure page.
func TestQASecureWAI(t *testing.T) {
	measurement, err := qaRunWithURL(
		"https://www.example.com",
		func(d *netem.DNSConfig) {
			qaAddTHDomains(d)
			qaAddExampleDomains(d)
		},
		func(d *netem.DPIEngine) {
			// nothing
		},
	)

	// fail the test on error
	if err != nil {
		t.Fatal(err)
	}

	// TODO(bassosimone,kelmenhorst): check the test keys
	_ = measurement
}
