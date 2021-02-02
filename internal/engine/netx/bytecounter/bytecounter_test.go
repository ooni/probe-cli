package bytecounter_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
)

func TestGood(t *testing.T) {
	counter := bytecounter.New()
	counter.CountBytesReceived(16384)
	counter.CountKibiBytesReceived(10)
	counter.CountBytesSent(2048)
	counter.CountKibiBytesSent(10)
	if counter.BytesSent() != 12288 {
		t.Fatal("invalid bytes sent")
	}
	if counter.BytesReceived() != 26624 {
		t.Fatal("invalid bytes received")
	}
	if v := counter.KibiBytesSent(); v < 11.9 || v > 12.1 {
		t.Fatal("invalid kibibytes sent")
	}
	if v := counter.KibiBytesReceived(); v < 25.9 || v > 26.1 {
		t.Fatal("invalid kibibytes received")
	}
}
