package httpclientx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestEndpoint(t *testing.T) {
	t.Run("the constructor only assigns the URL", func(t *testing.T) {
		epnt := NewEndpoint("https://www.example.com/")
		if epnt.URL != "https://www.example.com/" {
			t.Fatal("unexpected URL")
		}
		if epnt.Host != "" {
			t.Fatal("unexpected host")
		}
	})

	t.Run("we can optionally get a copy with an assigned host header", func(t *testing.T) {
		epnt := NewEndpoint("https://www.example.com/").WithHostOverride("www.cloudfront.com")
		if epnt.URL != "https://www.example.com/" {
			t.Fatal("unexpected URL")
		}
		if epnt.Host != "www.cloudfront.com" {
			t.Fatal("unexpected host")
		}
	})

	t.Run("we can convert from model.OOAPIService", func(t *testing.T) {
		services := []model.OOAPIService{{
			Address: "",
			Type:    "onion",
			Front:   "",
		}, {
			Address: "https://www.example.com/",
			Type:    "https",
		}, {
			Address: "",
			Type:    "onion",
			Front:   "",
		}, {
			Address: "https://www.example.com/",
			Type:    "cloudfront",
			Front:   "www.cloudfront.com",
		}, {
			Address: "",
			Type:    "onion",
			Front:   "",
		}}

		expect := []*Endpoint{{
			URL: "https://www.example.com/",
		}, {
			URL:  "https://www.example.com/",
			Host: "www.cloudfront.com",
		}}

		got := NewEndpointFromModelOOAPIServices(services...)
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})
}
