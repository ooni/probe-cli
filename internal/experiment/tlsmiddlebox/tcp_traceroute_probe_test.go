package tlsmiddlebox

import (
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestParseQuotedPacket(t *testing.T) {
	t.Run("buffer too short", func(t *testing.T) {
		buf := []byte("Hello")
		quotedPacket, err := parseQuotedPacket(buf)
		if quotedPacket == nil && (err == nil || err.Error() != "tcp quote too short") {
			t.Fatal("failed to error due to quote being too short")
		}
	})

	t.Run("success case", func(t *testing.T) {
		buf := []byte{0xed, 0x6e, 0x01, 0xbb, 0x16, 0xed, 0xf5, 0x4c}
		quotedPacket, err := parseQuotedPacket(buf)

		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		if quotedPacket == nil {
			t.Fatal("expected *model.ArchivalICMPQuotation, got nil")
		}

		if quotedPacket.Protocol != 6 {
			t.Fatalf("expected protocol 6, got %d", quotedPacket.Protocol)
		}

		if quotedPacket.SrcPort != 60782 {
			t.Fatalf("expected source port 60782, got %d", quotedPacket.SrcPort)
		}

		if quotedPacket.DstPort != 443 {
			t.Fatalf("expected destination port 443, got %d", quotedPacket.DstPort)
		}

		if quotedPacket.TCPSeqNum != 384693580 {
			t.Fatalf("expected TCP seq num 384693580, got %d", quotedPacket.TCPSeqNum)
		}
	})
}

func TestProbeTCP(t *testing.T) {
	wg := new(sync.WaitGroup)
	ttl := 2
	index := int64(1)

	t.Run("invalid IPv4 address", func(t *testing.T) {
		wg.Add(1)
		address := "298.125.34.4:443"
		ii, err := probeTCP(address, ttl, 3000, wg, model.DiscardLogger, index)
		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

		if err.Error() != "invalid IPv4 address" {
			t.Fatalf("expected 'invalid IPv4 address', got %s", err.Error())
		}

	})

	t.Run("invalid port number", func(t *testing.T) {
		wg.Add(1)
		address := "298.125.34.4:spot"
		ii, err := probeTCP(address, ttl, 3000, wg, model.DiscardLogger, index)
		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

		if err.Error() != "strconv.Atoi: parsing \"spot\": invalid syntax" {
			t.Fatalf("expected 'strconv.Atoi: parsing \"spot\": invalid syntax', got %s", err.Error())
		}
	})

}
