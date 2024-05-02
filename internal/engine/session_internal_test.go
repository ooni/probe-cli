package engine

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/checkincache"
	"github.com/ooni/probe-cli/v3/internal/enginelocate"
	"github.com/ooni/probe-cli/v3/internal/enginenetx"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/registry"
	"github.com/ooni/probe-cli/v3/internal/testingx"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func (s *Session) GetAvailableProbeServices() []model.OOAPIService {
	return s.getAvailableProbeServicesUnlocked()
}

func (s *Session) AppendAvailableProbeService(svc model.OOAPIService) {
	s.availableProbeServices = append(s.availableProbeServices, svc)
}

func (s *Session) QueryProbeServicesCount() int64 {
	return s.queryProbeServicesCount.Load()
}

// mockableProbeServicesClientForCheckIn allows us to mock the
// probeservices.Client used by Session.CheckIn.
type mockableProbeServicesClientForCheckIn struct {
	// Config is the config passed to the call.
	Config *model.OOAPICheckInConfig

	// Results contains the results of the call. This field MUST be
	// non-nil if and only if Error is nil.
	Results *model.OOAPICheckInResult

	// Error indicates whether the call failed. This field MUST be
	// non-nil if and only if Error is nil.
	Error error

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// CheckIn implements sessionProbeServicesClientForCheckIn.CheckIn.
func (c *mockableProbeServicesClientForCheckIn) CheckIn(
	ctx context.Context, config model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	defer c.mu.Unlock()
	c.mu.Lock()
	if c.Config != nil {
		return nil, errors.New("called more than once")
	}
	c.Config = &config
	if c.Results == nil && c.Error == nil {
		return nil, errors.New("misconfigured mockableProbeServicesClientForCheckIn")
	}
	return c.Results, c.Error
}

func TestSessionCheckInSuccessful(t *testing.T) {
	results := &model.OOAPICheckInResult{
		Tests: model.OOAPICheckInResultNettests{
			WebConnectivity: &model.OOAPICheckInInfoWebConnectivity{
				ReportID: "xxx-x-xx",
				URLs: []model.OOAPIURLInfo{{
					CategoryCode: "NEWS",
					CountryCode:  "IT",
					URL:          "https://www.repubblica.it/",
				}, {
					CategoryCode: "NEWS",
					CountryCode:  "IT",
					URL:          "https://www.unita.it/",
				}},
			},
		},
	}
	mockedClnt := &mockableProbeServicesClientForCheckIn{
		Results: results,
	}
	s := &Session{
		location: &enginelocate.Results{
			ASN:         137,
			CountryCode: "IT",
		},
		kvStore:         &kvstore.Memory{},
		softwareName:    "miniooni",
		softwareVersion: "0.1.0-dev",
		testMaybeLookupLocationContext: func(ctx context.Context) error {
			return nil
		},
		testNewProbeServicesClientForCheckIn: func(
			ctx context.Context) (sessionProbeServicesClientForCheckIn, error) {
			return mockedClnt, nil
		},
	}
	out, err := s.CheckIn(context.Background(), &model.OOAPICheckInConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if diff := cmp.Diff(results, out); diff != "" {
		t.Fatal(diff)
	}
	if mockedClnt.Config.Platform != s.Platform() {
		t.Fatal("invalid Config.Platform")
	}
	if mockedClnt.Config.ProbeASN != "AS137" {
		t.Fatal("invalid Config.ProbeASN")
	}
	if mockedClnt.Config.ProbeCC != "IT" {
		t.Fatal("invalid Config.ProbeCC")
	}
	if mockedClnt.Config.RunType != model.RunTypeTimed {
		t.Fatal("invalid Config.RunType")
	}
	if mockedClnt.Config.SoftwareName != "miniooni" {
		t.Fatal("invalid Config.SoftwareName")
	}
	if mockedClnt.Config.SoftwareVersion != "0.1.0-dev" {
		t.Fatal("invalid Config.SoftwareVersion")
	}
	if mockedClnt.Config.WebConnectivity.CategoryCodes == nil {
		t.Fatal("invalid ...CategoryCodes")
	}
}

func TestSessionCheckInNetworkError(t *testing.T) {
	expect := errors.New("mocked error")
	mockedClnt := &mockableProbeServicesClientForCheckIn{
		Error: expect,
	}
	s := &Session{
		location: &enginelocate.Results{
			ASN:         137,
			CountryCode: "IT",
		},
		softwareName:    "miniooni",
		softwareVersion: "0.1.0-dev",
		testMaybeLookupLocationContext: func(ctx context.Context) error {
			return nil
		},
		testNewProbeServicesClientForCheckIn: func(
			ctx context.Context) (sessionProbeServicesClientForCheckIn, error) {
			return mockedClnt, nil
		},
	}
	out, err := s.CheckIn(context.Background(), &model.OOAPICheckInConfig{})
	if !errors.Is(err, expect) {
		t.Fatal("unexpected err", err)
	}
	if out != nil {
		t.Fatal("expected nil out")
	}
}

func TestSessionCheckInCannotLookupLocation(t *testing.T) {
	errMocked := errors.New("mocked error")
	s := &Session{
		testMaybeLookupLocationContext: func(ctx context.Context) error {
			return errMocked
		},
	}
	out, err := s.CheckIn(context.Background(), &model.OOAPICheckInConfig{})
	if !errors.Is(err, errMocked) {
		t.Fatal("no the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil result here")
	}
}

func TestSessionCheckInCannotCreateProbeServicesClient(t *testing.T) {
	errMocked := errors.New("mocked error")
	s := &Session{
		location: &enginelocate.Results{
			ASN:         137,
			CountryCode: "IT",
		},
		softwareName:    "miniooni",
		softwareVersion: "0.1.0-dev",
		testMaybeLookupLocationContext: func(ctx context.Context) error {
			return nil
		},
		testNewProbeServicesClientForCheckIn: func(
			ctx context.Context) (sessionProbeServicesClientForCheckIn, error) {
			return nil, errMocked
		},
	}
	out, err := s.CheckIn(context.Background(), &model.OOAPICheckInConfig{})
	if !errors.Is(err, errMocked) {
		t.Fatal("no the error we expected", err)
	}
	if out != nil {
		t.Fatal("expected nil result here")
	}
}

func TestLowercaseMaybeLookupLocationContextWithCancelledContext(t *testing.T) {
	s := &Session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately kill the context
	err := s.maybeLookupLocationContext(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
}

func TestNewProbeServicesClientForCheckIn(t *testing.T) {
	s := &Session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately kill the context
	clnt, err := s.newProbeServicesClientForCheckIn(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if clnt != nil {
		t.Fatal("expected nil client here")
	}
}

func TestSessionNewSubmitterWithCancelledContext(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	sess := newSessionForTesting(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	subm, err := sess.NewSubmitter(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if subm != nil {
		t.Fatal("expected nil submitter here")
	}
}

func TestSessionMaybeLookupLocationContextLookupLocationContextFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	sess := newSessionForTestingNoLookups(t)
	sess.testLookupLocationContext = func(ctx context.Context) (*enginelocate.Results, error) {
		return nil, errMocked
	}
	err := sess.MaybeLookupLocationContext(context.Background())
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestSessionFetchTorTargetsWithCancelledContext(t *testing.T) {
	sess := &Session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	resp, err := sess.FetchTorTargets(ctx, "IT")
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestSessionFetchPsiphonConfigWithCancelledContext(t *testing.T) {
	sess := &Session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	resp, err := sess.FetchPsiphonConfig(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}

func TestNewSessionWithFakeTunnel(t *testing.T) {
	ctx := context.Background()
	sess, err := NewSession(ctx, SessionConfig{
		Logger:          log.Log,
		ProxyURL:        &url.URL{Scheme: "fake"},
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		TunnelDir:       "testdata",
	})
	if err != nil {
		t.Fatal(err)
	}
	if sess == nil {
		t.Fatal("expected non-nil session here")
	}
	if sess.ProxyURL() == nil {
		t.Fatal("expected non-nil proxyURL here")
	}
	if sess.tunnel == nil {
		t.Fatal("expected non-nil tunnel here")
	}
	sess.Close() // ensure we don't crash
}

func TestNewSessionWithFakeTunnelAndCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	sess, err := NewSession(ctx, SessionConfig{
		Logger:          log.Log,
		ProxyURL:        &url.URL{Scheme: "fake"},
		SoftwareName:    "miniooni",
		SoftwareVersion: "0.1.0-dev",
		TunnelDir:       "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if sess != nil {
		t.Fatal("expected nil session here")
	}
}

func TestSessionNewExperimentBuilder(t *testing.T) {
	t.Run("for a normal experiment", func(t *testing.T) {
		sess := &Session{
			logger: model.DiscardLogger,
		}
		builder, err := sess.NewExperimentBuilder("ndt7")
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment()
		if exp.Name() != "ndt" {
			t.Fatal("unexpected experiment")
		}
	})

	t.Run("for webconnectivity without feature flags", func(t *testing.T) {
		sess := &Session{
			kvStore: &kvstore.Memory{},
			logger:  model.DiscardLogger,
		}
		builder, err := sess.NewExperimentBuilder("web_connectivity")
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment()
		if exp.Name() != "web_connectivity" {
			t.Fatal("unexpected experiment")
		}
		switch m := exp.(*experiment).measurer; m.(type) {
		case webconnectivity.Measurer:
		default:
			t.Fatalf("unexpected measurer type: %T", m)
		}
	})

	t.Run("for webconnectivity with feature flags", func(t *testing.T) {
		memstore := &kvstore.Memory{}
		resp := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: map[string]bool{
					"webconnectivity_0.5": true,
				},
			},
			ProbeASN: "",
			ProbeCC:  "",
			Tests:    model.OOAPICheckInResultNettests{},
			UTCTime:  time.Time{},
			V:        0,
		}
		if err := checkincache.Store(memstore, resp); err != nil {
			t.Fatal(err)
		}
		sess := &Session{
			kvStore: memstore,
			logger:  model.DiscardLogger,
		}
		builder, err := sess.NewExperimentBuilder("web_connectivity")
		if err != nil {
			t.Fatal(err)
		}
		exp := builder.NewExperiment()
		if exp.Name() != "web_connectivity" {
			t.Fatal("unexpected experiment")
		}
		switch m := exp.(*experiment).measurer; m.(type) {
		case *webconnectivitylte.Measurer:
		default:
			t.Fatalf("unexpected measurer type %T", m)
		}
	})

	t.Run("for a nonexisting experiment", func(t *testing.T) {
		sess := &Session{
			logger: model.DiscardLogger,
		}
		builder, err := sess.NewExperimentBuilder("nonexistent")
		if !errors.Is(err, registry.ErrNoSuchExperiment) {
			t.Fatal("unexpected err", err)
		}
		if builder != nil {
			t.Fatal("expected nil builder here")
		}
	})
}

// This function tests the [*Session.CallWebConnectivityTestHelper] method.
func TestSessionCallWebConnectivityTestHelper(t *testing.T) {
	// We start with simple tests that exercise the basic functionality of the method
	// without bothering with having more than one available test helper.

	t.Run("when there are no available test helpers", func(t *testing.T) {
		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &Session{
			network: enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				model.DiscardLogger,
				nil,
				(&netxlite.Netx{}).NewStdlibResolver(model.DiscardLogger),
			),
			logger:          model.DiscardLogger,
			softwareName:    "miniooni",
			softwareVersion: version.Version,
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// invoke the API
		cresp, idx, err := sess.CallWebConnectivityTestHelper(ctx, creq, nil)

		// make sure we get the expected error
		if !errors.Is(err, model.ErrNoAvailableTestHelpers) {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is nil
		if cresp != nil {
			t.Fatal("expected nil, got", cresp)
		}
	})

	t.Run("when the call fails", func(t *testing.T) {
		// create a local test server that always resets the connection
		server := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &Session{
			network: enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				model.DiscardLogger,
				nil,
				(&netxlite.Netx{}).NewStdlibResolver(model.DiscardLogger),
			),
			logger:          model.DiscardLogger,
			softwareName:    "miniooni",
			softwareVersion: version.Version,
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := sess.CallWebConnectivityTestHelper(ctx, creq, testhelpers)

		// make sure we get the expected error
		if !errors.Is(err, netxlite.ECONNRESET) {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is nil
		if cresp != nil {
			t.Fatal("expected nil, got", cresp)
		}
	})

	t.Run("when the call succeeds", func(t *testing.T) {
		// create a local test server that always returns an ~empty response
		server := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &Session{
			network: enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				model.DiscardLogger,
				nil,
				(&netxlite.Netx{}).NewStdlibResolver(model.DiscardLogger),
			),
			logger:          model.DiscardLogger,
			softwareName:    "miniooni",
			softwareVersion: version.Version,
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := sess.CallWebConnectivityTestHelper(ctx, creq, testhelpers)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is zero
		if idx != 0 {
			t.Fatal("expected zero, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})

	t.Run("with two test helpers where the first one resets the connection and the second works", func(t *testing.T) {
		// create a local test server1 that always resets the connection
		server1 := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())
		defer server1.Close()

		// create a local test server2 that always returns an ~empty response
		server2 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server2.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &Session{
			network: enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				model.DiscardLogger,
				nil,
				(&netxlite.Netx{}).NewStdlibResolver(model.DiscardLogger),
			),
			logger:          model.DiscardLogger,
			softwareName:    "miniooni",
			softwareVersion: version.Version,
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server1.URL,
			Type:    "https",
			Front:   "",
		}, {
			Address: server2.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := sess.CallWebConnectivityTestHelper(ctx, creq, testhelpers)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is one
		if idx != 1 {
			t.Fatal("expected one, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})

	t.Run("with two test helpers where the first one times out the connection and the second works", func(t *testing.T) {
		// TODO(bassosimone): the utility of this test will become more obvious
		// once we switch this specific test to using httpclientx.

		// create a local test server1 that resets the connection after a ~long delay
		server1 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			select {
			case <-time.After(10 * time.Second):
				testingx.HTTPHandlerReset().ServeHTTP(w, r)
			case <-r.Context().Done():
				return
			}
		}))
		defer server1.Close()

		// create a local test server2 that always returns an ~empty response
		server2 := testingx.MustNewHTTPServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{}`))
		}))
		defer server2.Close()

		// create a new session only initializing the fields that
		// are going to matter for running this specific test
		sess := &Session{
			network: enginenetx.NewNetwork(
				bytecounter.New(),
				&kvstore.Memory{},
				model.DiscardLogger,
				nil,
				(&netxlite.Netx{}).NewStdlibResolver(model.DiscardLogger),
			),
			logger:          model.DiscardLogger,
			softwareName:    "miniooni",
			softwareVersion: version.Version,
		}

		// create a new background context
		ctx := context.Background()

		// create a fake request for the test helper
		//
		// note: no need to fill the request for this test case
		creq := &model.THRequest{}

		// create the list of test helpers to use
		testhelpers := []model.OOAPIService{{
			Address: server1.URL,
			Type:    "https",
			Front:   "",
		}, {
			Address: server2.URL,
			Type:    "https",
			Front:   "",
		}}

		// invoke the API
		cresp, idx, err := sess.CallWebConnectivityTestHelper(ctx, creq, testhelpers)

		// make sure we get the expected error
		if err != nil {
			t.Fatal("unexpected error", err)
		}

		// make sure idx is one
		if idx != 1 {
			t.Fatal("expected one, got", idx)
		}

		// make sure cresp is not nil
		if cresp == nil {
			t.Fatal("expected not nil, got", cresp)
		}
	})
}
