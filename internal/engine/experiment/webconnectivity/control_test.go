package webconnectivity_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
)

func TestFillASNsEmpty(t *testing.T) {
	dns := new(webconnectivity.ControlDNSResult)
	dns.FillASNs(new(mockable.Session))
	if diff := cmp.Diff(dns.ASNs, []int64{}); diff != "" {
		t.Fatal(diff)
	}
}

func TestFillASNsNoDatabase(t *testing.T) {
	dns := new(webconnectivity.ControlDNSResult)
	dns.Addrs = []string{"8.8.8.8", "1.1.1.1"}
	dns.FillASNs(new(mockable.Session))
	if diff := cmp.Diff(dns.ASNs, []int64{0, 0}); diff != "" {
		t.Fatal(diff)
	}
}

func TestFillASNsSuccess(t *testing.T) {
	sess := newsession(t, false)
	dns := new(webconnectivity.ControlDNSResult)
	dns.Addrs = []string{"8.8.8.8", "1.1.1.1"}
	dns.FillASNs(sess)
	if diff := cmp.Diff(dns.ASNs, []int64{15169, 13335}); diff != "" {
		t.Fatal(diff)
	}
}
