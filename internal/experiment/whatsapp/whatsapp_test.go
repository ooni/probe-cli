package whatsapp_test

import (
	"context"
	"fmt"
	"io"
	"regexp"
	"sync/atomic"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/experiment/whatsapp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := whatsapp.NewExperimentMeasurer(whatsapp.Config{})
	if measurer.ExperimentName() != "whatsapp" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.11.0" {
		t.Fatal("unexpected version")
	}
}

// whatsappWebAddr is the address of web.whatsapp.net and of v.whatsapp.net as of 2023-07-11
const whatsappWebAddr = "157.240.27.54"

// whatsappEndpointAddr is the address of whatsapp endpoints as of 2023-07-11
const whatsappEndpointAddr = "15.197.210.208"

// configureDNSForWeb creates DNS config for web.whatsapp.com using the given address
func configureDNSForWeb(config *netem.DNSConfig, addr string) {
	config.AddRecord("web.whatsapp.com", "web.whatsapp.com", addr)
}

// configureDNSForRegistrationService creates DNS config for v.whatsapp.net using the given address
func configureDNSForRegistrationService(config *netem.DNSConfig, addr string) {
	config.AddRecord("v.whatsapp.net", "v.whatsapp.net", addr)
}

// configureDNSForEndpoints creates DNS config for the endpoints using the given address
func configureDNSForEndpoints(config *netem.DNSConfig, addr string) {
	for idx := 1; idx <= 16; idx++ {
		config.AddRecord(
			fmt.Sprintf("e%d.whatsapp.net", idx),
			fmt.Sprintf("e%d.whatsapp.net", idx),
			addr,
		)
	}
}

// configureDNSWithDefaults creates DNS configuration using the default addresses
func configureDNSWithDefaults(config *netem.DNSConfig) {
	configureDNSForWeb(config, whatsappWebAddr)
	configureDNSForRegistrationService(config, whatsappWebAddr)
	configureDNSForEndpoints(config, whatsappEndpointAddr)
}

// newQAEnvironment creates a [*netemx.QAEnv] using the default configuration
func newQAEnvironment() *netemx.QAEnv {
	endpointsNetStack := netemx.QAEnvNetStackTCPEcho(log.Log, 443, 5222)

	// We need:
	//
	// - HTTPS listeners for whatsappWebAddr on port 443
	//
	// - TCP listeners for endpoints on 443 and 5222
	env := netemx.NewQAEnv(
		netemx.QAEnvOptionLogger(log.Log),
		netemx.QAEnvOptionHTTPServer(whatsappWebAddr, netemx.QAEnvDefaultHTTPHandler()),
		netemx.QAEnvOptionNetStack(whatsappEndpointAddr, endpointsNetStack),
	)

	// create default DNS configuration for all the existing resolvers, which specific nettests
	// will override in case they need to modify how the ISP's DNS resolver behaves
	configureDNSWithDefaults(env.ISPResolverConfig())
	configureDNSWithDefaults(env.OtherResolversConfig())

	return env
}

func TestMeasurerRun(t *testing.T) {
	t.Run("without DPI: expect success", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment()
		defer env.Close()

		env.Do(func() {
			measurer := whatsapp.NewExperimentMeasurer(whatsapp.Config{})
			sess := &mocks.Session{MockLogger: func() model.Logger { return log.Log }}
			measurement := new(model.Measurement)
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     sess,
			}

			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatal(err)
			}
			tk := measurement.TestKeys.(*whatsapp.TestKeys)
			if tk.RegistrationServerFailure != nil {
				t.Fatal("invalid RegistrationServerFailure")
			}
			if tk.RegistrationServerStatus != "ok" {
				t.Fatal("invalid RegistrationServerStatus")
			}
			if len(tk.WhatsappEndpointsBlocked) != 0 {
				t.Fatal("invalid WhatsappEndpointsBlocked")
			}
			if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
				t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
			}
			if tk.WhatsappEndpointsStatus != "ok" {
				t.Fatal("invalid WhatsappEndpointsStatus")
			}
			if tk.WhatsappWebFailure != nil {
				t.Fatal("invalid WhatsappWebFailure")
			}
			if tk.WhatsappWebStatus != "ok" {
				t.Fatal("invalid WhatsappWebStatus")
			}
		})
	})

	t.Run("with poisoned DNS: expect WhatsappWebFailure", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment()
		defer env.Close()

		// create DNS config with bogon entries for Whatsapp Web
		env.ISPResolverConfig().AddRecord("web.whatsapp.com", "web.whatsapp.com", "10.10.34.35")

		env.Do(func() {
			measurer := whatsapp.NewExperimentMeasurer(whatsapp.Config{})
			measurement := &model.Measurement{}
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*whatsapp.TestKeys)
			if tk.RegistrationServerFailure != nil {
				t.Fatal("invalid RegistrationServerFailure")
			}
			if tk.RegistrationServerStatus != "ok" {
				t.Fatal("invalid RegistrationServerStatus")
			}
			if len(tk.WhatsappEndpointsBlocked) != 0 {
				t.Fatal("invalid WhatsappEndpointsBlocked")
			}
			if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
				t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
			}
			if tk.WhatsappEndpointsStatus != "ok" {
				t.Fatal("invalid WhatsappEndpointsStatus")
			}
			if tk.WhatsappWebFailure == nil {
				t.Fatal("invalid WhatsappWebFailure")
			}
			if tk.WhatsappWebStatus == "ok" {
				t.Fatal("invalid WhatsappWebStatus")
			}
		})
	})

	t.Run("with DPI that drops TLS traffic with SNI = web.whatsapp.com: expect WhatsappWebFailure", func(t *testing.T) {
		// create a new test environment
		env := newQAEnvironment()
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "web.whatsapp.com",
		})

		env.Do(func() {
			measurer := whatsapp.NewExperimentMeasurer(whatsapp.Config{})
			measurement := &model.Measurement{}
			sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
			args := &model.ExperimentArgs{
				Callbacks:   model.NewPrinterCallbacks(log.Log),
				Measurement: measurement,
				Session:     sess,
			}
			err := measurer.Run(context.Background(), args)
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (measurement.TestKeys).(*whatsapp.TestKeys)
			if tk.RegistrationServerFailure != nil {
				t.Fatal("invalid RegistrationServerFailure")
			}
			if tk.RegistrationServerStatus != "ok" {
				t.Fatal("invalid RegistrationServerStatus")
			}
			if len(tk.WhatsappEndpointsBlocked) != 0 {
				t.Fatal("invalid WhatsappEndpointsBlocked")
			}
			if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
				t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
			}
			if tk.WhatsappEndpointsStatus != "ok" {
				t.Fatal("invalid WhatsappEndpointsStatus")
			}
			if tk.WhatsappWebFailure == nil {
				t.Fatal("invalid WhatsappWebFailure")
			}
			if tk.WhatsappWebStatus == "ok" {
				t.Fatal("invalid WhatsappWebStatus")
			}
		})
	})
}

func TestFailureAllEndpoints(t *testing.T) {
	// create a new test environment
	env := newQAEnvironment()
	defer env.Close()

	env.Do(func() {
		measurer := whatsapp.NewExperimentMeasurer(whatsapp.Config{})
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
		measurement := new(model.Measurement)
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		args := &model.ExperimentArgs{
			Callbacks:   callbacks,
			Measurement: measurement,
			Session:     sess,
		}
		err := measurer.Run(ctx, args)
		if err != nil {
			t.Fatal(err)
		}
		tk := measurement.TestKeys.(*whatsapp.TestKeys)
		if *tk.RegistrationServerFailure != "interrupted" {
			t.Fatal("invalid RegistrationServerFailure")
		}
		if tk.RegistrationServerStatus != "blocked" {
			t.Fatal("invalid RegistrationServerStatus")
		}
		if len(tk.WhatsappEndpointsBlocked) != 16 {
			t.Fatal("invalid WhatsappEndpointsBlocked")
		}
		pattern := regexp.MustCompile("^e[0-9]{1,2}.whatsapp.net$")
		for i := 0; i < len(tk.WhatsappEndpointsBlocked); i++ {
			if !pattern.MatchString(tk.WhatsappEndpointsBlocked[i]) {
				t.Fatalf("invalid WhatsappEndpointsBlocked[%d]", i)
			}
		}
		if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
			t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
		}
		if tk.WhatsappEndpointsStatus != "blocked" {
			t.Fatal("invalid WhatsappEndpointsStatus")
		}
		if *tk.WhatsappWebFailure != "interrupted" {
			t.Fatal("invalid WhatsappWebFailure")
		}
		if tk.WhatsappWebStatus != "blocked" {
			t.Fatal("invalid WhatsappWebStatus")
		}
		sk, err := measurer.GetSummaryKeys(measurement)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := sk.(whatsapp.SummaryKeys); !ok {
			t.Fatal("invalid type for summary keys")
		}
	})
}

func TestTestKeysComputeWebStatus(t *testing.T) {
	errorString := io.EOF.Error()
	type fields struct {
		TestKeys                         urlgetter.TestKeys
		RegistrationServerFailure        *string
		RegistrationServerStatus         string
		WhatsappEndpointsBlocked         []string
		WhatsappEndpointsDNSInconsistent []string
		WhatsappEndpointsStatus          string
		WhatsappWebStatus                string
		WhatsappWebFailure               *string
		WhatsappHTTPSFailure             *string
	}
	tests := []struct {
		name    string
		fields  fields
		failure *string
		status  string
	}{{
		name:    "with success",
		failure: nil,
		status:  "ok",
	}, {
		name: "with HTTPS failure",
		fields: fields{
			WhatsappHTTPSFailure: &errorString,
		},
		failure: &errorString,
		status:  "blocked",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tk := &whatsapp.TestKeys{
				TestKeys:                         tt.fields.TestKeys,
				RegistrationServerFailure:        tt.fields.RegistrationServerFailure,
				RegistrationServerStatus:         tt.fields.RegistrationServerStatus,
				WhatsappEndpointsBlocked:         tt.fields.WhatsappEndpointsBlocked,
				WhatsappEndpointsDNSInconsistent: tt.fields.WhatsappEndpointsDNSInconsistent,
				WhatsappEndpointsStatus:          tt.fields.WhatsappEndpointsStatus,
				WhatsappWebStatus:                tt.fields.WhatsappWebStatus,
				WhatsappWebFailure:               tt.fields.WhatsappWebFailure,
				WhatsappHTTPSFailure:             tt.fields.WhatsappHTTPSFailure,
			}
			tk.ComputeWebStatus()
			diff := cmp.Diff(tk.WhatsappWebFailure, tt.failure)
			if diff != "" {
				t.Fatal(diff)
			}
			diff = cmp.Diff(tk.WhatsappWebStatus, tt.status)
			if diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestTestKeysMixedEndpointsFailure(t *testing.T) {
	failure := io.EOF.Error()
	tk := whatsapp.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:443"},
		TestKeys: urlgetter.TestKeys{Failure: &failure},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:5222"},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.RegistrationServiceURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.WebHTTPSURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.ComputeWebStatus()
	if tk.RegistrationServerFailure != nil {
		t.Fatal("invalid RegistrationServerFailure")
	}
	if tk.RegistrationServerStatus != "ok" {
		t.Fatal("invalid RegistrationServerStatus")
	}
	if len(tk.WhatsappEndpointsBlocked) != 0 {
		t.Fatal("invalid WhatsappEndpointsBlocked")
	}
	if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
		t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
	}
	if tk.WhatsappEndpointsStatus != "ok" {
		t.Fatal("invalid WhatsappEndpointsStatus")
	}
	if tk.WhatsappWebFailure != nil {
		t.Fatal("invalid WhatsappWebFailure")
	}
	if tk.WhatsappWebStatus != "ok" {
		t.Fatal("invalid WhatsappWebStatus")
	}
}

func TestTestKeysOnlyEndpointsFailure(t *testing.T) {
	failure := io.EOF.Error()
	tk := whatsapp.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:443"},
		TestKeys: urlgetter.TestKeys{Failure: &failure},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:5222"},
		TestKeys: urlgetter.TestKeys{Failure: &failure},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.RegistrationServiceURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.WebHTTPSURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.ComputeWebStatus()
	if tk.RegistrationServerFailure != nil {
		t.Fatal("invalid RegistrationServerFailure")
	}
	if tk.RegistrationServerStatus != "ok" {
		t.Fatal("invalid RegistrationServerStatus")
	}
	if len(tk.WhatsappEndpointsBlocked) != 1 {
		t.Fatal("invalid WhatsappEndpointsBlocked")
	}
	if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
		t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
	}
	if tk.WhatsappEndpointsStatus != "blocked" {
		t.Fatal("invalid WhatsappEndpointsStatus")
	}
	if tk.WhatsappWebFailure != nil {
		t.Fatal("invalid WhatsappWebFailure")
	}
	if tk.WhatsappWebStatus != "ok" {
		t.Fatal("invalid WhatsappWebStatus")
	}
}

func TestTestKeysOnlyRegistrationServerFailure(t *testing.T) {
	failure := io.EOF.Error()
	tk := whatsapp.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:443"},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.RegistrationServiceURL},
		TestKeys: urlgetter.TestKeys{Failure: &failure},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.WebHTTPSURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.ComputeWebStatus()
	if *tk.RegistrationServerFailure != failure {
		t.Fatal("invalid RegistrationServerFailure")
	}
	if tk.RegistrationServerStatus != "blocked" {
		t.Fatal("invalid RegistrationServerStatus")
	}
	if len(tk.WhatsappEndpointsBlocked) != 0 {
		t.Fatal("invalid WhatsappEndpointsBlocked")
	}
	if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
		t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
	}
	if tk.WhatsappEndpointsStatus != "ok" {
		t.Fatal("invalid WhatsappEndpointsStatus")
	}
	if tk.WhatsappWebFailure != nil {
		t.Fatal("invalid WhatsappWebFailure")
	}
	if tk.WhatsappWebStatus != "ok" {
		t.Fatal("invalid WhatsappWebStatus")
	}
}

func TestTestKeysOnlyWebHTTPSFailure(t *testing.T) {
	failure := io.EOF.Error()
	tk := whatsapp.NewTestKeys()
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: "tcpconnect://e7.whatsapp.net:443"},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.RegistrationServiceURL},
		TestKeys: urlgetter.TestKeys{},
	})
	tk.Update(urlgetter.MultiOutput{
		Input:    urlgetter.MultiInput{Target: whatsapp.WebHTTPSURL},
		TestKeys: urlgetter.TestKeys{Failure: &failure},
	})
	tk.ComputeWebStatus()
	if tk.RegistrationServerFailure != nil {
		t.Fatal("invalid RegistrationServerFailure")
	}
	if tk.RegistrationServerStatus != "ok" {
		t.Fatal("invalid RegistrationServerStatus")
	}
	if len(tk.WhatsappEndpointsBlocked) != 0 {
		t.Fatal("invalid WhatsappEndpointsBlocked")
	}
	if len(tk.WhatsappEndpointsDNSInconsistent) != 0 {
		t.Fatal("invalid WhatsappEndpointsDNSInconsistent")
	}
	if tk.WhatsappEndpointsStatus != "ok" {
		t.Fatal("invalid WhatsappEndpointsStatus")
	}
	if *tk.WhatsappWebFailure != failure {
		t.Fatal("invalid WhatsappWebFailure")
	}
	if tk.WhatsappWebStatus != "blocked" {
		t.Fatal("invalid WhatsappWebStatus")
	}
}

func TestWeConfigureWebChecksCorrectly(t *testing.T) {
	called := &atomic.Int64{}
	emptyConfig := urlgetter.Config{}
	measurer := whatsapp.Measurer{
		Config: whatsapp.Config{},
		Getter: func(ctx context.Context, g urlgetter.Getter) (urlgetter.TestKeys, error) {
			switch g.Target {
			case whatsapp.WebHTTPSURL:
				called.Add(1)
				if diff := cmp.Diff(g.Config, emptyConfig); diff != "" {
					panic(diff)
				}
			case whatsapp.RegistrationServiceURL:
				called.Add(4)
				if diff := cmp.Diff(g.Config, emptyConfig); diff != "" {
					panic(diff)
				}
			default:
				called.Add(8)
				if diff := cmp.Diff(g.Config, emptyConfig); diff != "" {
					panic(diff)
				}
			}
			return urlgetter.DefaultMultiGetter(ctx, g)
		},
	}

	ctx := context.Background()
	sess := &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }}
	measurement := new(model.Measurement)
	callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err := measurer.Run(ctx, args); err != nil {
		t.Fatal(err)
	}
	const expected = 261
	if got := called.Load(); got != expected {
		t.Fatalf("not called the expected number of times: expected = %d, got = %d", expected, got)
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &whatsapp.Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysWorksAsIntended(t *testing.T) {
	tests := []struct {
		tk                         whatsapp.TestKeys
		RegistrationServerBlocking bool
		WebBlocking                bool
		EndpointsBlocking          bool
		isAnomaly                  bool
	}{{
		tk:                         whatsapp.TestKeys{},
		RegistrationServerBlocking: false,
		WebBlocking:                false,
		EndpointsBlocking:          false,
		isAnomaly:                  false,
	}, {
		tk: whatsapp.TestKeys{
			RegistrationServerStatus: "blocked",
		},
		RegistrationServerBlocking: true,
		WebBlocking:                false,
		EndpointsBlocking:          false,
		isAnomaly:                  true,
	}, {
		tk: whatsapp.TestKeys{
			WhatsappWebStatus: "blocked",
		},
		RegistrationServerBlocking: false,
		WebBlocking:                true,
		EndpointsBlocking:          false,
		isAnomaly:                  true,
	}, {
		tk: whatsapp.TestKeys{
			WhatsappEndpointsStatus: "blocked",
		},
		RegistrationServerBlocking: false,
		WebBlocking:                false,
		EndpointsBlocking:          true,
		isAnomaly:                  true,
	}}
	for idx, tt := range tests {
		t.Run(fmt.Sprintf("%d", idx), func(t *testing.T) {
			m := &whatsapp.Measurer{}
			measurement := &model.Measurement{TestKeys: &tt.tk}
			got, err := m.GetSummaryKeys(measurement)
			if err != nil {
				t.Fatal(err)
				return
			}
			sk := got.(whatsapp.SummaryKeys)
			if sk.IsAnomaly != tt.isAnomaly {
				t.Fatal("unexpected isAnomaly value")
			}
			if sk.RegistrationServerBlocking != tt.RegistrationServerBlocking {
				t.Fatal("unexpected registrationServerBlocking value")
			}
			if sk.WebBlocking != tt.WebBlocking {
				t.Fatal("unexpected webBlocking value")
			}
			if sk.EndpointsBlocking != tt.EndpointsBlocking {
				t.Fatal("unexpected endpointsBlocking value")
			}
		})
	}
}
