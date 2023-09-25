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

	cases := []testcase{
		{-1, 0}, // make sure we gracefully handle negative numbers (i.e., we don't crash)
		{0, 0},
		{1, time.Second},
		{2, 2 * time.Second},
		{3, 4 * time.Second},
		{4, 8 * time.Second},
		{5, 2 * 8 * time.Second},
		{6, 3 * 8 * time.Second},
		{7, 4 * 8 * time.Second},
		{8, 5 * 8 * time.Second},
		{9, 6 * 8 * time.Second},
		{10, 7 * 8 * time.Second},
	}

	for _, tc := range cases {
		t.Run(fmt.Sprintf("tc.idx=%v", tc.idx), func(t *testing.T) {
			got := happyEyeballsDelay(tc.idx)
			if got != tc.expect {
				t.Fatalf("with tc.idx=%v we got %v but expected %v", tc.idx, got, tc.expect)
			}
			t.Logf("with tc.idx=%v: got %v", tc.idx, got)
		})
	}
}
