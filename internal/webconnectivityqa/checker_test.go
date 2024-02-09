package webconnectivityqa_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivitylte"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/webconnectivityqa"
)

func TestConfigureCustomCheckers(t *testing.T) {
	tc := &webconnectivityqa.TestCase{
		Name:      "",
		Input:     "",
		ExpectErr: false,
		ExpectTestKeys: &webconnectivityqa.TestKeys{
			Accessible: true,
			Blocking:   nil,
		},
		Checkers: []webconnectivityqa.Checker{&webconnectivityqa.ReadWriteEventsExistentialChecker{}},
	}
	measurer := &mocks.ExperimentMeasurer{
		MockExperimentName: func() string {
			return "web_connectivity"
		},
		MockExperimentVersion: func() string {
			return "0.5.28"
		},
		MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
			args.Measurement.TestKeys = &webconnectivitylte.TestKeys{
				Accessible: optional.Some(true),
				Blocking:   nil,
			}
			return nil
		},
	}
	err := webconnectivityqa.RunTestCase(measurer, tc)
	if !errors.Is(err, webconnectivityqa.ErrCheckerNoReadWriteEvents) {
		t.Fatal("unexpected error", err)
	}
}

func TestReadWriteEventsExistentialChecker(t *testing.T) {
	type testcase struct {
		name    string
		version string
		tk      string
		expect  error
	}

	cases := []testcase{{
		name:    "with Web Connectivity v0.4",
		version: "0.4.3",
		tk:      `{}`,
		expect:  nil,
	}, {
		name:    "with Web Connectivity v0.6",
		version: "0.6.0",
		tk:      `{}`,
		expect:  webconnectivityqa.ErrCheckerUnexpectedWebConnectivityVersion,
	}, {
		name:    "with read/write network events",
		version: "0.5.28",
		tk:      `{"network_events":[{"operation":"read"},{"operation":"write"}]}`,
		expect:  nil,
	}, {
		name:    "without network events",
		version: "0.5.28",
		tk:      `{"network_events":[]}`,
		expect:  webconnectivityqa.ErrCheckerNoReadWriteEvents,
	}, {
		name:    "with no read/write network events",
		version: "0.5.28",
		tk:      `{"network_events":[{"operation":"connect"},{"operation":"close"}]}`,
		expect:  webconnectivityqa.ErrCheckerNoReadWriteEvents,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var tks map[string]any
			must.UnmarshalJSON([]byte(tc.tk), &tks)

			meas := &model.Measurement{
				TestKeys:    tks,
				TestVersion: tc.version,
			}

			err := (&webconnectivityqa.ReadWriteEventsExistentialChecker{}).Check(meas)

			switch {
			case tc.expect == nil && err == nil:
				return

			case tc.expect == nil && err != nil:
				t.Fatal("expected", tc.expect, "got", err)

			case tc.expect != nil && err == nil:
				t.Fatal("expected", tc.expect, "got", err)

			case tc.expect != nil && err != nil:
				if err.Error() != tc.expect.Error() {
					t.Fatal("expected", tc.expect, "got", err)
				}
			}
		})
	}
}
