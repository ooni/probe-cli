package hhfm

import (
	"net/http"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func TestNewHeadersFromMap(t *testing.T) {

	// testcase is a test case run by this func
	type testcase struct {
		name   string
		input  map[string]string
		expect map[string][]string
	}

	cases := []testcase{{
		name:   "with nil input",
		input:  nil,
		expect: http.Header{},
	}, {
		name:   "with empty input",
		input:  map[string]string{},
		expect: http.Header{},
	}, {
		name: "common case: headers with mixed casing should be preserved",
		input: map[string]string{
			"ConTent-TyPe": "text/html; charset=utf-8",
			"ViA":          "a",
			"User-AgeNt":   "miniooni/0.1.0",
		},
		expect: map[string][]string{
			"ConTent-TyPe": {"text/html; charset=utf-8"},
			"ViA":          {"a"},
			"User-AgeNt":   {"miniooni/0.1.0"},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := newHeadersFromMap(tc.input)
			if diff := cmp.Diff(tc.expect, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
