package websites

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/nettests"
	"github.com/pkg/errors"
)

// URLInfo contains the URL and the citizenlab category code for that URL
type URLInfo struct {
	URL          string `json:"url"`
	CategoryCode string `json:"category_code"`
}

// URLResponse is the orchestrate url response containing a list of URLs
type URLResponse struct {
	Results []URLInfo `json:"results"`
}

const orchestrateBaseURL = "https://events.proteus.test.ooni.io"

func lookupURLs(ctl *nettests.Controller) ([]string, error) {
	var (
		parsed = new(URLResponse)
		urls   []string
	)
	reqURL := fmt.Sprintf("%s/api/v1/urls?probe_cc=%s",
		orchestrateBaseURL,
		ctl.Ctx.Location.CountryCode)

	resp, err := http.Get(reqURL)
	if err != nil {
		return urls, errors.Wrap(err, "failed to perform request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return urls, errors.Wrap(err, "failed to read response body")
	}
	err = json.Unmarshal([]byte(body), &parsed)
	if err != nil {
		return urls, errors.Wrap(err, "failed to parse json")
	}

	for _, url := range parsed.Results {
		urls = append(urls, url.URL)
	}
	return urls, nil
}

// WebConnectivity test implementation
type WebConnectivity struct {
}

// Run starts the test
func (n WebConnectivity) Run(ctl *nettests.Controller) error {
	nt := mk.NewNettest("WebConnectivity")
	ctl.Init(nt)

	urls, err := lookupURLs(ctl)
	if err != nil {
		return err
	}
	nt.Options.Inputs = urls

	return nt.Run()
}

// WebConnectivitySummary for the test
type WebConnectivitySummary struct {
	Accessible bool
	Blocking   string
	Blocked    bool
}

// Summary generates a summary for a test run
func (n WebConnectivity) Summary(tk map[string]interface{}) interface{} {
	var (
		blocked    bool
		blocking   string
		accessible bool
	)

	// We need to do these complicated type assertions, because some of the fields
	// are "nullable" and/or can be of different types
	switch v := tk["blocking"].(type) {
	case bool:
		blocked = false
		blocking = "none"
	case string:
		blocked = true
		blocking = v
	default:
		blocked = false
		blocking = "none"
	}

	if tk["accessible"] == nil {
		accessible = false
	} else {
		accessible = tk["accessible"].(bool)
	}

	return WebConnectivitySummary{
		Accessible: accessible,
		Blocking:   blocking,
		Blocked:    blocked,
	}
}

// LogSummary writes the summary to the standard output
func (n WebConnectivity) LogSummary(s string) error {
	return nil
}
