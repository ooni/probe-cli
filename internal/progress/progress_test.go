package progress

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type capturerCallbacks struct {
	value float64
}

var _ model.ExperimentCallbacks = &capturerCallbacks{}

// OnProgress implements model.ExperimentCallbacks.
func (v *capturerCallbacks) OnProgress(percentage float64, message string) {
	v.value = percentage
}

func TestScaler(t *testing.T) {
	// testcase is a test case run by this function.
	type testcase struct {
		// name is the test case name.
		name string

		// offset is the offset (>=0, <total)
		offset float64

		// total is the total (>0, <=1)
		total float64

		// emit is the list of progress values to emit.
		emit []float64

		// expect is the list of progress values we expect in output.
		expect []float64
	}

	cases := []testcase{{
		name:   "with offset==0 and total=1",
		offset: 0,
		total:  1,
		emit:   []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
		expect: []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
	}, {
		name:   "with offset==0 and total=0.5",
		offset: 0,
		total:  0.5,
		emit:   []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
		expect: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5},
	}, {
		name:   "with offset==0.5 and total=1",
		offset: 0.5,
		total:  1,
		emit:   []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
		expect: []float64{0.5, 0.6, 0.7, 0.8, 0.9, 1},
	}, {
		name:   "with offset==0.2 and total=0.7",
		offset: 0.2,
		total:  0.7,
		emit:   []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
		expect: []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7},
	}, {
		name:   "with offset=0.4 and total=0.5",
		offset: 0.4,
		total:  0.5,
		emit:   []float64{0, 0.2, 0.4, 0.6, 0.8, 1},
		expect: []float64{0.4, 0.42, 0.44, 0.46, 0.48, 0.5},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var got []float64
			for _, v := range tc.emit {
				cc := &capturerCallbacks{}
				wrapper := NewScaler(cc, tc.offset, tc.total)
				wrapper.OnProgress(v, "")
				got = append(got, cc.value)
			}
			if diff := cmp.Diff(tc.expect, got, cmpopts.EquateApprox(0, 0.01)); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
