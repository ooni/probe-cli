// Package whatsapp contains the WhatsApp network experiment.
//
// See https://github.com/ooni/spec/blob/master/nettests/ts-018-whatsapp.md.
package whatsapp

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"regexp"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpfailure"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const (
	// RegistrationServiceURL is the URL used by WhatsApp registration service
	RegistrationServiceURL = "https://v.whatsapp.net/v2/register"

	// WebHTTPURL is WhatsApp web's HTTP URL
	WebHTTPURL = "http://web.whatsapp.com/"

	// WebHTTPSURL is WhatsApp web's HTTPS URL
	WebHTTPSURL = "https://web.whatsapp.com/"

	testName    = "whatsapp"
	testVersion = "0.9.0"
)

var endpointPattern = regexp.MustCompile(`^tcpconnect://e[0-9]{1,2}\.whatsapp\.net:[0-9]{3,5}$`)

// Config contains the experiment config.
type Config struct{}

// TestKeys contains the experiment results
type TestKeys struct {
	urlgetter.TestKeys
	RegistrationServerFailure        *string        `json:"registration_server_failure"`
	RegistrationServerStatus         string         `json:"registration_server_status"`
	WhatsappEndpointsBlocked         []string       `json:"whatsapp_endpoints_blocked"`
	WhatsappEndpointsDNSInconsistent []string       `json:"whatsapp_endpoints_dns_inconsistent"`
	WhatsappEndpointsStatus          string         `json:"whatsapp_endpoints_status"`
	WhatsappWebFailure               *string        `json:"whatsapp_web_failure"`
	WhatsappWebStatus                string         `json:"whatsapp_web_status"`
	WhatsappEndpointsCount           map[string]int `json:"-"`
	WhatsappHTTPFailure              *string        `json:"-"`
	WhatsappHTTPSFailure             *string        `json:"-"`
}

// NewTestKeys returns a new instance of the test keys.
func NewTestKeys() *TestKeys {
	failure := "unknown_failure"
	return &TestKeys{
		RegistrationServerFailure:        &failure,
		RegistrationServerStatus:         "blocked",
		WhatsappEndpointsBlocked:         []string{},
		WhatsappEndpointsDNSInconsistent: []string{},
		WhatsappEndpointsStatus:          "blocked",
		WhatsappWebFailure:               &failure,
		WhatsappWebStatus:                "blocked",
		WhatsappEndpointsCount:           make(map[string]int),
		WhatsappHTTPFailure:              &failure,
		WhatsappHTTPSFailure:             &failure,
	}
}

// Update updates the TestKeys using the given MultiOutput result.
func (tk *TestKeys) Update(v urlgetter.MultiOutput) {
	// Update the easy to update entries first
	tk.NetworkEvents = append(tk.NetworkEvents, v.TestKeys.NetworkEvents...)
	tk.Queries = append(tk.Queries, v.TestKeys.Queries...)
	tk.Requests = append(tk.Requests, v.TestKeys.Requests...)
	tk.TCPConnect = append(tk.TCPConnect, v.TestKeys.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, v.TestKeys.TLSHandshakes...)
	// Set the status of WhatsApp endpoints
	if endpointPattern.MatchString(v.Input.Target) {
		if v.TestKeys.Failure != nil {
			parsed, err := url.Parse(v.Input.Target)
			runtimex.PanicOnError(err, "url.Parse should not fail here")
			hostname := parsed.Hostname()
			tk.WhatsappEndpointsCount[hostname]++
			if tk.WhatsappEndpointsCount[hostname] >= 2 {
				tk.WhatsappEndpointsBlocked = append(tk.WhatsappEndpointsBlocked, hostname)
			}
			return
		}
		tk.WhatsappEndpointsStatus = "ok"
		return
	}
	// Set the status of the registration service
	if v.Input.Target == RegistrationServiceURL {
		tk.RegistrationServerFailure = v.TestKeys.Failure
		if v.TestKeys.Failure == nil {
			tk.RegistrationServerStatus = "ok"
		}
		return
	}
	// Track result of accessing the web interface.
	switch v.Input.Target {
	case WebHTTPSURL:
		tk.WhatsappHTTPSFailure = v.TestKeys.Failure
	case WebHTTPURL:
		failure := v.TestKeys.Failure
		if failure != nil {
			// nothing to do here
		} else if v.TestKeys.HTTPResponseStatus != 302 {
			failure = &httpfailure.UnexpectedStatusCode
		} else if len(v.TestKeys.HTTPResponseLocations) != 1 {
			failure = &httpfailure.UnexpectedRedirectURL
		} else if v.TestKeys.HTTPResponseLocations[0] != WebHTTPSURL {
			failure = &httpfailure.UnexpectedRedirectURL
		}
		tk.WhatsappHTTPFailure = failure
	}
}

// ComputeWebStatus sets the web status fields.
func (tk *TestKeys) ComputeWebStatus() {
	if tk.WhatsappHTTPFailure == nil && tk.WhatsappHTTPSFailure == nil {
		tk.WhatsappWebFailure = nil
		tk.WhatsappWebStatus = "ok"
		return
	}
	tk.WhatsappWebStatus = "blocked" // must be here because of unit tests
	if tk.WhatsappHTTPSFailure != nil {
		tk.WhatsappWebFailure = tk.WhatsappHTTPSFailure
		return
	}
	tk.WhatsappWebFailure = tk.WhatsappHTTPFailure
}

// Measurer performs the measurement
type Measurer struct {
	// Config contains the experiment settings. If empty we
	// will be using default settings.
	Config Config

	// Getter is an optional getter to be used for testing.
	Getter urlgetter.MultiGetter
}

// ExperimentName implements ExperimentMeasurer.ExperimentName
func (m Measurer) ExperimentName() string {
	return testName
}

// ExperimentVersion implements ExperimentMeasurer.ExperimentVersion
func (m Measurer) ExperimentVersion() string {
	return testVersion
}

// Run implements ExperimentMeasurer.Run
func (m Measurer) Run(
	ctx context.Context, sess model.ExperimentSession,
	measurement *model.Measurement, callbacks model.ExperimentCallbacks,
) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	urlgetter.RegisterExtensions(measurement)
	// generate all the inputs
	var inputs []urlgetter.MultiInput
	for idx := 1; idx <= 16; idx++ {
		for _, port := range []string{"443", "5222"} {
			inputs = append(inputs, urlgetter.MultiInput{
				Target: fmt.Sprintf("tcpconnect://e%d.whatsapp.net:%s", idx, port),
			})
		}
	}
	inputs = append(inputs, urlgetter.MultiInput{
		Config: urlgetter.Config{FailOnHTTPError: true},
		Target: RegistrationServiceURL,
	})
	inputs = append(inputs, urlgetter.MultiInput{
		// We consider this check successful if we can establish a TLS
		// connection and we don't see any socket/TLS errors. Hence, we
		// don't care about the HTTP response code.
		Target: WebHTTPSURL,
	})
	inputs = append(inputs, urlgetter.MultiInput{
		// We consider this check successful if we get a valid redirect
		// for the HTTPS web interface. No need to follow redirects.
		Config: urlgetter.Config{NoFollowRedirects: true},
		Target: WebHTTPURL,
	})
	rnd := rand.New(rand.NewSource(time.Now().UnixNano()))
	rnd.Shuffle(len(inputs), func(i, j int) {
		inputs[i], inputs[j] = inputs[j], inputs[i]
	})
	// measure in parallel
	multi := urlgetter.Multi{Begin: time.Now(), Getter: m.Getter, Session: sess}
	testkeys := NewTestKeys()
	testkeys.Agent = "redirect"
	measurement.TestKeys = testkeys
	for entry := range multi.Collect(ctx, inputs, "whatsapp", callbacks) {
		testkeys.Update(entry)
	}
	testkeys.ComputeWebStatus()
	return nil
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
	RegistrationServerBlocking bool `json:"registration_server_blocking"`
	WebBlocking                bool `json:"whatsapp_web_blocking"`
	EndpointsBlocking          bool `json:"whatsapp_endpoints_blocking"`
	IsAnomaly                  bool `json:"-"`
}

// GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys.
func (m Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	sk := SummaryKeys{IsAnomaly: false}
	tk, ok := measurement.TestKeys.(*TestKeys)
	if !ok {
		return sk, errors.New("invalid test keys type")
	}
	blocking := func(value string) bool {
		return value == "blocked"
	}
	sk.RegistrationServerBlocking = blocking(tk.RegistrationServerStatus)
	sk.WebBlocking = blocking(tk.WhatsappWebStatus)
	sk.EndpointsBlocking = blocking(tk.WhatsappEndpointsStatus)
	sk.IsAnomaly = (sk.RegistrationServerBlocking || sk.WebBlocking || sk.EndpointsBlocking)
	return sk, nil
}
