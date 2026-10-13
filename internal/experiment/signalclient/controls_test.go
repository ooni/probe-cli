package signalclient

import (
	"strings"
	"testing"
)

func TestDecoySNI(t *testing.T) {
	for _, host := range []string{"grpc.chat.signal.org", "cdn.signal.org", "a.b"} {
		d := decoySNI(len(host))
		if !strings.HasSuffix(d, ".example.com") || (len(host) > len(".example.com") && len(d) != len(host)) {
			t.Errorf("%s: %s", host, d)
		}
		if strings.ToLower(d) != d {
			t.Errorf("not lowercase: %s", d)
		}
	}
}
