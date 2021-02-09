package oonimkall_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/pkg/oonimkall"
)

func TestSessionWebConnectivity(t *testing.T) {
	sess, err := NewSession()
	if err != nil {
		t.Fatal(err)
	}
	ctx := sess.NewContext()
	config := &oonimkall.WebConnectivityConfig{
		Input: "https://www.google.com",
	}
	results, err := sess.WebConnectivity(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("bytes received: %f", results.KibiBytesReceived)
	t.Logf("bytes sent: %f", results.KibiBytesSent)
	t.Logf("measurement: %d bytes", len(results.Measurement))
}
