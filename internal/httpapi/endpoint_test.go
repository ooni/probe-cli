package httpapi

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewEndpointList(t *testing.T) {
	type args struct {
		httpClient model.HTTPClient
		userAgent  string
		services   []model.OOAPIService
	}
	defaultHTTPClient := &mocks.HTTPClient{}
	tests := []struct {
		name    string
		args    args
		wantOut []*Endpoint
	}{{
		name: "with no services",
		args: args{
			httpClient: defaultHTTPClient,
			userAgent:  model.HTTPHeaderUserAgent,
			services:   nil,
		},
		wantOut: nil,
	}, {
		name: "common cases",
		args: args{
			httpClient: defaultHTTPClient,
			userAgent:  model.HTTPHeaderUserAgent,
			services: []model.OOAPIService{{
				Address: "https://www.example.com/",
				Type:    "https",
				Front:   "",
			}, {
				Address: "https://www.example.org/",
				Type:    "cloudfront",
				Front:   "example.org.it",
			}, {
				Address: "https://nonexistent.onion/",
				Type:    "onion",
				Front:   "",
			}},
		},
		wantOut: []*Endpoint{{
			BaseURL:    "https://www.example.com/",
			HTTPClient: defaultHTTPClient,
			Host:       "",
			Logger:     model.DiscardLogger,
			UserAgent:  model.HTTPHeaderUserAgent,
		}, {
			BaseURL:    "https://www.example.org/",
			HTTPClient: defaultHTTPClient,
			Host:       "example.org.it",
			Logger:     model.DiscardLogger,
			UserAgent:  model.HTTPHeaderUserAgent,
		}},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := NewEndpointList(
				tt.args.httpClient,
				model.DiscardLogger,
				tt.args.userAgent,
				tt.args.services...,
			)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
