package enginenetx

import (
	"fmt"
	"testing"
	"time"
)

func TestHappyEyeballsDelay(t *testing.T) {
	type testcase struct {
		idx    int
		expect time.Duration
	}

	const delay = 900 * time.Millisecond

	cases := []testcase{
		{-1, 0}, // make sure we gracefully handle negative numbers (i.e., we don't crash)
		{0, 0},
		{1, delay},
		{2, delay * 2},
		{3, delay * 4},
		{4, delay * 8},
		{5, delay * 16},
		{6, 15 * time.Second},
		{7, 30 * time.Second},
		{8, 45 * time.Second},
		{9, 60 * time.Second},
		{10, 75 * time.Second},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("delay=%v tc.idx=%v", delay, tc.idx), func(t *testing.T) {
			got := happyEyeballsDelay(delay, tc.idx)
			if got != tc.expect {
				t.Fatalf("with delay=%v tc.idx=%v we got %v but expected %v", delay, tc.idx, got, tc.expect)
			}
			t.Logf("with delay=%v tc.idx=%v: got %v", delay, tc.idx, got)
		})
	}
}
