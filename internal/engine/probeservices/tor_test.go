package probeservices_test

import (
	"context"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices/testorchestra"
)

func TestFetchTorTargets(t *testing.T) {
	clnt := newclient()
	if err := clnt.MaybeRegister(context.Background(), testorchestra.MetadataFixture()); err != nil {
		t.Fatal(err)
	}
	if err := clnt.MaybeLogin(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := clnt.FetchTorTargets(context.Background(), "ZZ")
	if err != nil {
		t.Fatal(err)
	}
	if data == nil || len(data) <= 0 {
		t.Fatal("invalid data")
	}
}

func TestFetchTorTargetsNotRegistered(t *testing.T) {
	clnt := newclient()
	state := probeservices.State{
		// Explicitly empty so the test is more clear
	}
	if err := clnt.StateFile.Set(state); err != nil {
		t.Fatal(err)
	}
	data, err := clnt.FetchTorTargets(context.Background(), "ZZ")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if data != nil {
		t.Fatal("expected nil data here")
	}
}

type FetchTorTargetsHTTPTransport struct {
	Response *http.Response
}

func (clnt *FetchTorTargetsHTTPTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	if req.URL.Path == "/api/v1/test-list/tor-targets" {
		clnt.Response = resp
	}
	return resp, err
}

func TestFetchTorTargetsSetsQueryString(t *testing.T) {
	clnt := newclient()
	txp := new(FetchTorTargetsHTTPTransport)
	clnt.HTTPClient.Transport = txp
	if err := clnt.MaybeRegister(context.Background(), testorchestra.MetadataFixture()); err != nil {
		t.Fatal(err)
	}
	if err := clnt.MaybeLogin(context.Background()); err != nil {
		t.Fatal(err)
	}
	data, err := clnt.FetchTorTargets(context.Background(), "ZZ")
	if err != nil {
		t.Fatal(err)
	}
	if data == nil || len(data) <= 0 {
		t.Fatal("invalid data")
	}
	requestURL := txp.Response.Request.URL
	if requestURL.Query().Get("country_code") != "ZZ" {
		t.Fatal("invalid country code")
	}
}
