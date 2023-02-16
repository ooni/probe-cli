package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestMainWorkingAsIntended(t *testing.T) {
	// let the kernel pick a random free port
	*apiEndpoint = "127.0.0.1:0"

	// run the main function in a background goroutine
	go main()

	// prepare the HTTP request body
	jsonReq := ctrlRequest{
		HTTPRequest: "https://dns.google",
		HTTPRequestHeaders: map[string][]string{
			"Accept":          {model.HTTPHeaderAccept},
			"Accept-Language": {model.HTTPHeaderAcceptLanguage},
			"User-Agent":      {model.HTTPHeaderUserAgent},
		},
		TCPConnect: []string{
			"8.8.8.8:443",
			"8.8.4.4:443",
		},
	}
	data, err := json.Marshal(jsonReq)
	runtimex.PanicOnError(err, "cannot marshal request")

	// construct the test helper's URL
	endpoint := <-srvAddr
	URL := &url.URL{
		Scheme: "http",
		Host:   endpoint,
		Path:   "/",
	}
	req, err := http.NewRequest("POST", URL.String(), bytes.NewReader(data))
	runtimex.PanicOnError(err, "cannot create new HTTP request")

	// issue the request and get the response
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("unexpected status code", resp.StatusCode)
	}

	// read the response body
	data, err = netxlite.ReadAllContext(context.Background(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	// parse the response
	var jsonResp ctrlResponse
	if err := json.Unmarshal(data, &jsonResp); err != nil {
		t.Fatal(err)
	}

	// very simple correctness check
	if !strings.Contains(jsonResp.HTTPRequest.Title, "Google") {
		t.Fatal("expected the response title to contain the string Google")
	}

	// tear down the TH
	sigs <- syscall.SIGINT

	// wait for the background goroutine to join
	srvWg.Wait()
}
