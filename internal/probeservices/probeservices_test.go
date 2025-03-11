package probeservices

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func newclient() *Client {
	client, err := NewClient(
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		model.OOAPIService{
			Address: "https://api.dev.ooni.io/",
			Type:    "https",
		},
	)
	if err != nil {
		panic(err) // so fail the test
	}
	return client
}

func TestNewClientHTTPS(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://x.org",
			Type:    "https",
		})
	if err != nil {
		t.Fatal(err)
	}
	if client.BaseURL != "https://x.org" {
		t.Fatal("not the URL we expected")
	}
}

func TestNewClientUnsupportedService(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://x.org",
			Type:    "onion",
		})
	if !errors.Is(err, ErrUnsupportedServiceType) {
		t.Fatal("not the error we expected")
	}
	if client != nil {
		t.Fatal("expected nil client here")
	}
}

func TestNewClientCloudfrontInvalidURL(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "\t\t\t",
			Type:    "cloudfront",
		})
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if client != nil {
		t.Fatal("expected nil client here")
	}
}

func TestNewClientCloudfrontInvalidURLScheme(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "http://x.org",
			Type:    "cloudfront",
		})
	if !errors.Is(err, ErrUnsupportedCloudFrontAddress) {
		t.Fatal("not the error we expected")
	}
	if client != nil {
		t.Fatal("expected nil client here")
	}
}

func TestNewClientCloudfrontInvalidURLWithPort(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://x.org:54321",
			Type:    "cloudfront",
		})
	if !errors.Is(err, ErrUnsupportedCloudFrontAddress) {
		t.Fatal("not the error we expected")
	}
	if client != nil {
		t.Fatal("expected nil client here")
	}
}

func TestNewClientCloudfrontInvalidFront(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://x.org",
			Type:    "cloudfront",
			Front:   "\t\t\t",
		})
	if err == nil || !strings.HasSuffix(err.Error(), `invalid URL escape "%09"`) {
		t.Fatal("not the error we expected")
	}
	if client != nil {
		t.Fatal("expected nil client here")
	}
}

func TestNewClientCloudfrontGood(t *testing.T) {
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://x.org",
			Type:    "cloudfront",
			Front:   "google.com",
		})
	if err != nil {
		t.Fatal(err)
	}
	if client.BaseURL != "https://google.com" {
		t.Fatal("not the BaseURL we expected")
	}
	if client.Host != "x.org" {
		t.Fatal("not the Host we expected")
	}
}

func TestCloudfront(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	client, err := NewClient(
		&mockable.Session{}, model.OOAPIService{
			Address: "https://meek.azureedge.net",
			Type:    "cloudfront",
			Front:   "ajax.aspnetcdn.com",
		})
	if err != nil {
		t.Fatal(err)
	}
	req, err := http.NewRequest("GET", client.BaseURL, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Host = client.Host
	resp, err := http.DefaultTransport.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatal("unexpected status code")
	}
	data, err := netxlite.ReadAllContext(req.Context(), resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "I’m just a happy little web server.\n" {
		t.Fatal("unexpected response body")
	}
}

func TestDefaultProbeServicesWorkAsIntended(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	for _, e := range Default() {
		client, err := NewClient(&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		}, e)
		if err != nil {
			t.Fatal(err)
		}
		testhelpers, err := client.GetTestHelpers(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if len(testhelpers) < 1 {
			t.Fatal("no test helpers?!")
		}
	}
}

func TestSortServices(t *testing.T) {
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "https",
		Address: "https://ams-ps2.ooni.nu:443",
	}}
	expect := []model.OOAPIService{{
		Type:    "https",
		Address: "https://ams-ps2.ooni.nu:443",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}}
	out := SortServices(in)
	diff := cmp.Diff(out, expect)
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestOnlyHTTPS(t *testing.T) {
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Type:    "https",
		Address: "https://ams-ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://hkg-ps-nonexistent.ooni.io",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "https",
		Address: "https://mia-ps-nonexistent.ooni.io",
	}}
	expect := []model.OOAPIService{{
		Type:    "https",
		Address: "https://ams-ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://hkg-ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://mia-ps-nonexistent.ooni.io",
	}}
	out := OnlyHTTPS(in)
	diff := cmp.Diff(out, expect)
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestOnlyFallbacks(t *testing.T) {
	// put onion first so we also verify that we sort the services
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Type:    "https",
		Address: "https://ams-ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://hkg-ps-nonexistent.ooni.io",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "https",
		Address: "https://mia-ps-nonexistent.ooni.io",
	}}
	expect := []model.OOAPIService{{
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}}
	out := OnlyFallbacks(in)
	diff := cmp.Diff(out, expect)
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestTryAllCanceledContext(t *testing.T) {
	// put onion first so we also verify that we sort the services
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Type:    "https",
		Address: "https://ams-ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://hkg-ps-nonexistent.ooni.io",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "https",
		Address: "https://mia-ps-nonexistent.ooni.io",
	}}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel and cause every attempt to fail
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
	}
	out := TryAll(ctx, sess, in)
	if len(out) != 5 {
		t.Fatal("invalid number of entries")
	}
	//
	if out[0].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !errors.Is(out[0].Err, context.Canceled) {
		t.Fatal("invalid error")
	}
	if out[0].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[0].Service.Address != "https://ams-ps-nonexistent.ooni.io" {
		t.Fatal("invalid service type")
	}
	//
	if out[1].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !errors.Is(out[1].Err, context.Canceled) {
		t.Fatal("invalid error")
	}
	if out[1].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[1].Service.Address != "https://hkg-ps-nonexistent.ooni.io" {
		t.Fatal("invalid service type")
	}
	//
	if out[2].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !errors.Is(out[2].Err, context.Canceled) {
		t.Fatal("invalid error")
	}
	if out[2].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[2].Service.Address != "https://mia-ps-nonexistent.ooni.io" {
		t.Fatal("invalid service type")
	}
	//
	if out[3].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !errors.Is(out[3].Err, context.Canceled) {
		t.Fatal("invalid error")
	}
	if out[3].Service.Type != "cloudfront" {
		t.Fatal("invalid service type")
	}
	if out[3].Service.Front != "dkyhjv0wpi2dk.cloudfront.net" {
		t.Fatal("invalid service type")
	}
	if out[3].Service.Address != "https://dkyhjv0wpi2dk.cloudfront.net" {
		t.Fatal("invalid service type")
	}
	//
	// Note: here duration may be zero because the service is not supported
	// and so we don't basically do anything. But it also may be nonzero since
	// we also run tests in the cloud, which is slower than my desktop. So, I
	// have not written a specific test concerning out[4].Duration.
	if !errors.Is(out[4].Err, ErrUnsupportedServiceType) {
		t.Fatal("invalid error")
	}
	if out[4].Service.Type != "onion" {
		t.Fatal("invalid service type")
	}
	if out[4].Service.Address != "httpo://jehhrikjjqrlpufu.onion" {
		t.Fatal("invalid service type")
	}
}

func TestTryAllIntegrationWeRaceForFastestHTTPS(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// put onion first so we also verify that we sort the services
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Type:    "https",
		Address: "https://api.ooni.io",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}}
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
	}
	out := TryAll(context.Background(), sess, in)
	if len(out) != 1 {
		t.Fatal("invalid number of entries")
	}
	//
	if out[0].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if out[0].Err != nil {
		t.Fatal("invalid error")
	}
	if out[0].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[0].Service.Address != "https://api.ooni.io" {
		t.Fatal("invalid service address")
	}
}

func TestTryAllIntegrationWeFallback(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// put onion first so we also verify that we sort the services
	in := []model.OOAPIService{{
		Type:    "onion",
		Address: "httpo://jehhrikjjqrlpufu.onion",
	}, {
		Type:    "https",
		Address: "https://ps-nonexistent.ooni.io",
	}, {
		Type:    "https",
		Address: "https://hkg-ps-nonexistent.ooni.nu",
	}, {
		Front:   "dkyhjv0wpi2dk.cloudfront.net",
		Type:    "cloudfront",
		Address: "https://dkyhjv0wpi2dk.cloudfront.net",
	}, {
		Type:    "https",
		Address: "https://mia-ps2-nonexistent.ooni.nu",
	}}
	sess := &mockable.Session{
		MockableHTTPClient: http.DefaultClient,
		MockableLogger:     log.Log,
	}
	out := TryAll(context.Background(), sess, in)
	if len(out) != 4 {
		t.Fatal("invalid number of entries")
	}
	//
	if out[0].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !strings.HasSuffix(out[0].Err.Error(), "no such host") {
		t.Fatal("invalid error")
	}
	if out[0].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[0].Service.Address != "https://ps-nonexistent.ooni.io" {
		t.Fatal("invalid service type")
	}
	//
	if out[1].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !strings.HasSuffix(out[1].Err.Error(), "no such host") {
		t.Fatal("invalid error")
	}
	if out[1].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[1].Service.Address != "https://hkg-ps-nonexistent.ooni.nu" {
		t.Fatal("invalid service type")
	}
	//
	if out[2].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if !strings.HasSuffix(out[2].Err.Error(), "no such host") {
		t.Fatal("invalid error")
	}
	if out[2].Service.Type != "https" {
		t.Fatal("invalid service type")
	}
	if out[2].Service.Address != "https://mia-ps2-nonexistent.ooni.nu" {
		t.Fatal("invalid service type")
	}
	//
	if out[3].Duration <= 0 {
		t.Fatal("invalid duration")
	}
	if out[3].Err != nil {
		t.Fatal("invalid error")
	}
	if out[3].Service.Type != "cloudfront" {
		t.Fatal("invalid service type")
	}
	if out[3].Service.Address != "https://dkyhjv0wpi2dk.cloudfront.net" {
		t.Fatal("invalid service type")
	}
	if out[3].Service.Front != "dkyhjv0wpi2dk.cloudfront.net" {
		t.Fatal("invalid front")
	}
}

func TestSelectBestEmptyInput(t *testing.T) {
	if out := SelectBest(nil); out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestSelectBestOnlyFailures(t *testing.T) {
	in := []*Candidate{{
		Duration: 10 * time.Millisecond,
		Err:      io.EOF,
	}}
	if out := SelectBest(in); out != nil {
		t.Fatal("expected nil output here")
	}
}

func TestSelectBestSelectsTheFastest(t *testing.T) {
	in := []*Candidate{{
		Duration: 10 * time.Millisecond,
		Service: model.OOAPIService{
			Address: "https://ps1.ooni.nonexistent",
			Type:    "https",
		},
	}, {
		Duration: 4 * time.Millisecond,
		Service: model.OOAPIService{
			Address: "https://ps2.ooni.nonexistent",
			Type:    "https",
		},
	}, {
		Duration: 7 * time.Millisecond,
		Service: model.OOAPIService{
			Address: "https://ps3.ooni.nonexistent",
			Type:    "https",
		},
	}, {
		Duration: 11 * time.Millisecond,
		Service: model.OOAPIService{
			Address: "https://ps4.ooni.nonexistent",
			Type:    "https",
		},
	}}
	expected := &Candidate{
		Duration: 4 * time.Millisecond,
		Service: model.OOAPIService{
			Address: "https://ps2.ooni.nonexistent",
			Type:    "https",
		},
	}
	out := SelectBest(in)
	if diff := cmp.Diff(out, expected); diff != "" {
		t.Fatal(diff)
	}
}

func TestGetCredsAndAuthNotLoggedIn(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	clnt := newclient()
	if err := clnt.MaybeRegister(context.Background(), MetadataFixture()); err != nil {
		t.Fatal(err)
	}
	creds, auth, err := clnt.GetCredsAndAuth()
	if !errors.Is(err, ErrNotLoggedIn) {
		t.Fatal("not the error we expected")
	}
	if creds != nil {
		t.Fatal("expected nil creds here")
	}
	if auth != nil {
		t.Fatal("expected nil auth here")
	}
}
