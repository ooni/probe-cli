package mocks

import (
	"context"
	"errors"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestSession(t *testing.T) {
	t.Run("GetTestHelpersByName", func(t *testing.T) {
		var expect []model.OOAPIService
		ff := &testingx.FakeFiller{}
		ff.Fill(&expect)
		runtimex.Assert(len(expect) > 0, "expected non-empty array")
		s := &Session{
			MockGetTestHelpersByName: func(name string) ([]model.OOAPIService, bool) {
				return expect, len(expect) > 0
			},
		}
		out, good := s.GetTestHelpersByName("xx")
		if !good {
			t.Fatal("not good")
		}
		if diff := cmp.Diff(expect, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("DefaultHTTPClient", func(t *testing.T) {
		expected := &HTTPClient{}
		s := &Session{
			MockDefaultHTTPClient: func() model.HTTPClient {
				return expected
			},
		}
		out := s.DefaultHTTPClient()
		if expected != out {
			t.Fatal("unexpected result")
		}
	})

	t.Run("FetchPsiphonConfig", func(t *testing.T) {
		var expected []byte
		ff := &testingx.FakeFiller{}
		ff.Fill(&expected)
		runtimex.Assert(len(expected) > 0, "expected nonempty list")
		s := &Session{
			MockFetchPsiphonConfig: func(ctx context.Context) ([]byte, error) {
				return expected, nil
			},
		}
		out, err := s.FetchPsiphonConfig(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expected, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("FetchTorTargets", func(t *testing.T) {
		expected := errors.New("mocked err")
		s := &Session{
			MockFetchTorTargets: func(ctx context.Context, cc string) (map[string]model.OOAPITorTarget, error) {
				return nil, expected
			},
		}
		out, err := s.FetchTorTargets(context.Background(), "IT")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if len(out) > 0 {
			t.Fatal("expected empty out")
		}
	})

	t.Run("KeyValueStore", func(t *testing.T) {
		expect := &KeyValueStore{}
		s := &Session{
			MockKeyValueStore: func() model.KeyValueStore {
				return expect
			},
		}
		out := s.KeyValueStore()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("Logger", func(t *testing.T) {
		expect := &Logger{}
		s := &Session{
			MockLogger: func() model.Logger {
				return expect
			},
		}
		out := s.Logger()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("MaybeResolverIP", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockMaybeResolverIP: func() string {
				return expect
			},
		}
		out := s.MaybeResolverIP()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ProbeASNString", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockProbeASNString: func() string {
				return expect
			},
		}
		out := s.ProbeASNString()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ProbeCC", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockProbeCC: func() string {
				return expect
			},
		}
		out := s.ProbeCC()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ProbeIP", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockProbeIP: func() string {
				return expect
			},
		}
		out := s.ProbeIP()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ProbeNetworkName", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockProbeNetworkName: func() string {
				return expect
			},
		}
		out := s.ProbeNetworkName()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ProxyURL", func(t *testing.T) {
		expect := &url.URL{Scheme: "xx"}
		s := &Session{
			MockProxyURL: func() *url.URL {
				return expect
			},
		}
		out := s.ProxyURL()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("ResolverIP", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockResolverIP: func() string {
				return expect
			},
		}
		out := s.ResolverIP()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("SoftwareName", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockSoftwareName: func() string {
				return expect
			},
		}
		out := s.SoftwareName()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("SoftwareVersion", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockSoftwareVersion: func() string {
				return expect
			},
		}
		out := s.SoftwareVersion()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("TempDir", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockTempDir: func() string {
				return expect
			},
		}
		out := s.TempDir()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("TorArgs", func(t *testing.T) {
		var expect []string
		ff := &testingx.FakeFiller{}
		ff.Fill(&expect)
		runtimex.Assert(len(expect) > 0, "expected non empty slice")
		s := &Session{
			MockTorArgs: func() []string {
				return expect
			},
		}
		out := s.TorArgs()
		if diff := cmp.Diff(expect, out); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("TorBinary", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockTorBinary: func() string {
				return expect
			},
		}
		out := s.TorBinary()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("TunnelDir", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockTunnelDir: func() string {
				return expect
			},
		}
		out := s.TunnelDir()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("UserAgent", func(t *testing.T) {
		expect := "xx"
		s := &Session{
			MockUserAgent: func() string {
				return expect
			},
		}
		out := s.UserAgent()
		if out != expect {
			t.Fatal("invalid output")
		}
	})

	t.Run("NewExperimentBuilder", func(t *testing.T) {
		eb := &ExperimentBuilder{}
		s := &Session{
			MockNewExperimentBuilder: func(name string) (model.ExperimentBuilder, error) {
				return eb, nil
			},
		}
		out, err := s.NewExperimentBuilder("x")
		if err != nil {
			t.Fatal(err)
		}
		if out != eb {
			t.Fatal("invalid output")
		}
	})

	t.Run("NewSubmitter", func(t *testing.T) {
		expected := errors.New("mocked err")
		s := &Session{
			MockNewSubmitter: func(ctx context.Context) (model.Submitter, error) {
				return nil, expected
			},
		}
		out, err := s.NewSubmitter(context.Background())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err")
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})

	t.Run("CheckIn", func(t *testing.T) {
		expected := errors.New("mocked err")
		s := &Session{
			MockCheckIn: func(ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInNettests, error) {
				return nil, expected
			},
		}
		out, err := s.CheckIn(context.Background(), &model.OOAPICheckInConfig{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err")
		}
		if out != nil {
			t.Fatal("unexpected out")
		}
	})
}
