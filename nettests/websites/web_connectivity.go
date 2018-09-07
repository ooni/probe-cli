package websites

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/apex/log"
	"github.com/measurement-kit/go-measurement-kit"
	"github.com/ooni/probe-cli/internal/database"
	"github.com/ooni/probe-cli/nettests"
	"github.com/pkg/errors"
)

// URLInfo contains the URL and the citizenlab category code for that URL
type URLInfo struct {
	URL          string `json:"url"`
	CountryCode  string `json:"country_code"`
	CategoryCode string `json:"category_code"`
}

// URLResponse is the orchestrate url response containing a list of URLs
type URLResponse struct {
	Results []URLInfo `json:"results"`
}

const orchestrateBaseURL = "https://events.proteus.test.ooni.io"

func lookupURLs(ctl *nettests.Controller) ([]string, map[int64]int64, error) {
	var (
		parsed   = new(URLResponse)
		urls     []string
		urlIDMap map[int64]int64
	)
	// XXX pass in the configuration for category codes
	reqURL := fmt.Sprintf("%s/api/v1/urls?probe_cc=%s",
		orchestrateBaseURL,
		ctl.Ctx.Location.CountryCode)

	resp, err := http.Get(reqURL)
	if err != nil {
		return urls, urlIDMap, errors.Wrap(err, "failed to perform request")
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return urls, urlIDMap, errors.Wrap(err, "failed to read response body")
	}
	err = json.Unmarshal([]byte(body), &parsed)
	if err != nil {
		return urls, urlIDMap, errors.Wrap(err, "failed to parse json")
	}

	for idx, url := range parsed.Results {
		var urlID int64

		res, err := ctl.Ctx.DB.Update("urls").Set(
			"url", url.URL,
			"category_code", url.CategoryCode,
			"country_code", url.CountryCode,
		).Where("url = ? AND country_code = ?", url.URL, url.CountryCode).Exec()

		if err != nil {
			log.Error("Failed to write to the URL table")
		} else {
			affected, err := res.RowsAffected()

			if err != nil {
				log.Error("Failed to get affected row count")
			} else if affected == 0 {
				newID, err := ctl.Ctx.DB.Collection("urls").Insert(
					database.URL{
						URL:          url.URL,
						CategoryCode: url.CategoryCode,
						CountryCode:  url.CountryCode,
					})
				if err != nil {
					log.Error("Failed to insert into the URLs table")
				}
				urlID = newID.(int64)
			} else {
				lastID, err := res.LastInsertId()
				if err != nil {
					log.Error("failed to get URL ID")
				}
				urlID = lastID
			}
		}

		urlIDMap[int64(idx)] = urlID
		urls = append(urls, url.URL)
	}
	return urls, urlIDMap, nil
}

// WebConnectivity test implementation
type WebConnectivity struct {
}

// Run starts the test
func (n WebConnectivity) Run(ctl *nettests.Controller) error {
	nt := mk.NewNettest("WebConnectivity")
	ctl.Init(nt)

	urls, urlIDMap, err := lookupURLs(ctl)
	if err != nil {
		return err
	}
	ctl.SetInputIdxMap(urlIDMap)
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
