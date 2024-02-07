package webconnectivitylte

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

func TestSummaryKeys(t *testing.T) {
	type testcase struct {
		name             string
		accessible       optional.Value[bool]
		blocking         any
		expectAccessible bool
		expectBlocking   string
		expectIsAnomaly  bool
	}

	cases := []testcase{{
		name:             "with all nil",
		accessible:       optional.None[bool](),
		blocking:         nil,
		expectAccessible: false,
		expectBlocking:   "",
		expectIsAnomaly:  false,
	}, {
		name:             "with success",
		accessible:       optional.Some(true),
		blocking:         false,
		expectAccessible: true,
		expectBlocking:   "",
		expectIsAnomaly:  false,
	}, {
		name:             "with website down",
		accessible:       optional.Some(false),
		blocking:         false,
		expectAccessible: false,
		expectBlocking:   "",
		expectIsAnomaly:  false,
	}, {
		name:             "with anomaly",
		accessible:       optional.Some(false),
		blocking:         "http-failure",
		expectAccessible: false,
		expectBlocking:   "http-failure",
		expectIsAnomaly:  true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tk := &TestKeys{
				Accessible: tc.accessible,
				Blocking:   tc.blocking,
			}
			sk := tk.MeasurementSummaryKeys().(*SummaryKeys)
			if sk.Accessible != tc.expectAccessible {
				t.Fatal("expected", tc.expectAccessible, "got", sk.Accessible)
			}
			if sk.Blocking != tc.expectBlocking {
				t.Fatal("expected", tc.expectBlocking, "got", sk.Blocking)
			}
			if sk.IsAnomaly != tc.expectIsAnomaly {
				t.Fatal("expected", tc.expectIsAnomaly, "got", sk.IsAnomaly)
			}
			if sk.Anomaly() != sk.IsAnomaly {
				t.Fatal("expected Anomaly() to equal IsAnomaly")
			}
		})
	}
}
