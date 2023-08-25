package webconnectivity

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestFillASNsEmpty(t *testing.T) {
	dns := new(ControlDNSResult)
	sess := &mocks.Session{
		MockLookupASN: geoipx.LookupASN,
	}
	fillASNs(sess, dns)
	if diff := cmp.Diff(dns.ASNs, []int64{}); diff != "" {
		t.Fatal(diff)
	}
}

func TestFillASNsSuccess(t *testing.T) {
	dns := new(ControlDNSResult)
	dns.Addrs = []string{"8.8.8.8", "1.1.1.1"}
	sess := &mocks.Session{
		MockLookupASN: geoipx.LookupASN,
	}
	fillASNs(sess, dns)
	if diff := cmp.Diff(dns.ASNs, []int64{15169, 13335}); diff != "" {
		t.Fatal(diff)
	}
}
