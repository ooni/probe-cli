package engine

import (
	"context"
	"errors"
	"net/url"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/geolocate"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func (s *Session) GetAvailableProbeServices() []model.Service {
	return s.getAvailableProbeServicesUnlocked()
}

func (s *Session) AppendAvailableProbeService(svc model.Service) {
	s.availableProbeServices = append(s.availableProbeServices, svc)
}

func (s *Session) QueryProbeServicesCount() int64 {
	return s.queryProbeServicesCount.Load()
}

// mockableProbeServicesClientForCheckIn allows us to mock the
// probeservices.Client used by Session.CheckIn.
type mockableProbeServicesClientForCheckIn struct {
	// Config is the config passed to the call.
	Config *model.CheckInConfig

	// Results contains the results of the call. This field MUST be
	// non-nil if and only if Error is nil.
	Results *model.CheckInInfo

	// Error indicates whether the call failed. This field MUST be
	// non-nil if and only if Error is nil.
	Error error

	// mu provides mutual exclusion.
	mu sync.Mutex
}

// CheckIn implements sessionProbeServicesClientForCheckIn.CheckIn.
func (c *mockableProbeServicesClientForCheckIn) CheckIn(
	ctx context.Context, config model.CheckInConfig) (*model.CheckInInfo, error) {
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
	results := &model.CheckInInfo{
		WebConnectivity: &model.CheckInInfoWebConnectivity{
			ReportID: "xxx-x-xx",
			URLs: []model.URLInfo{{
				CategoryCode: "NEWS",
				CountryCode:  "IT",
				URL:          "https://www.repubblica.it/",
			}, {
				CategoryCode: "NEWS",
				CountryCode:  "IT",
				URL:          "https://www.unita.it/",
			}},
		},
	}
	mockedClnt := &mockableProbeServicesClientForCheckIn{
		Results: results,
	}
	s := &Session{
		location: &geolocate.Results{
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
	out, err := s.CheckIn(context.Background(), &model.CheckInConfig{})
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
	if mockedClnt.Config.RunType != "timed" {
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

func TestSessionCheckInCannotLookupLocation(t *testing.T) {
	errMocked := errors.New("mocked error")
	s := &Session{
		testMaybeLookupLocationContext: func(ctx context.Context) error {
			return errMocked
		},
	}
	out, err := s.CheckIn(context.Background(), &model.CheckInConfig{})
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
		location: &geolocate.Results{
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
	out, err := s.CheckIn(context.Background(), &model.CheckInConfig{})
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

func TestSessionMaybeLookupLocationContextLookupLocationContextFailure(t *testing.T) {
	errMocked := errors.New("mocked error")
	sess := newSessionForTestingNoLookups(t)
	sess.testLookupLocationContext = func(ctx context.Context) (*geolocate.Results, error) {
		return nil, errMocked
	}
	err := sess.MaybeLookupLocationContext(context.Background())
	if !errors.Is(err, errMocked) {
		t.Fatal("not the error we expected", err)
	}
}

func TestSessionFetchURLListWithCancelledContext(t *testing.T) {
	sess := &Session{}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	resp, err := sess.FetchURLList(ctx, model.URLListConfig{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response here")
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
