package tlsmiddlebox

import (
	"context"
	"errors"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestScrubIPsFromError(t *testing.T) {
	t.Run("scrub IPv4 address", func(t *testing.T) {
		ipv4String := "I like to play 67.89.34.92:90 with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed] with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv4 address with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play 123.21.56.102:443: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub bogon IPv4 address with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play 10.2.0.219:443: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address", func(t *testing.T) {
		ipv4String := "I like to play [2001:db8:abcd:0012::1]:90 with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed] with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address loopback with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play [::1]:8080: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address short with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play [2001:db8::1]:443: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address long with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play [2001:0db8:0000:abC5:face:78bE:0000:0001]:443: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address link local with trailing colon", func(t *testing.T) {
		ipv4String := "I like to play [fe80::1]:9090: with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed]: with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address link local without trailing colon and brackets", func(t *testing.T) {
		ipv4String := "I like to play fe80::1 with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed] with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})

	t.Run("scrub IPv6 address without trailing colon and brackets", func(t *testing.T) {
		ipv4String := "I like to play 2001:db8::1 with maracas"

		ipv4StringScrubbed := scrubIPsFromError(ipv4String)

		ipv4StringResult := "I like to play [scrubbed] with maracas"

		if strings.Compare(ipv4StringScrubbed, ipv4StringResult) != 0 {
			t.Fatalf("unexpected ipv4StringScrubbed, expected %s, got %s", ipv4StringResult, ipv4StringScrubbed)
		}
	})
}

func TestNewIterationFromHandshake(t *testing.T) {
	t.Run("with an error and no handshake", func(t *testing.T) {
		iter := newIterationFromHandshake(3, errors.New("mocked error"), nil, nil, "example.com")
		if iter.TTL != 3 {
			t.Fatal("unexpected TTL value")
		}
		if iter.Handshake == nil || iter.Handshake.ServerName != "example.com" {
			t.Fatal("unexpected servername")
		}
		if iter.Handshake.Failure == nil || *iter.Handshake.Failure != "unknown_failure: mocked error" {
			t.Fatal("unexpected failure", iter.Handshake.Failure)
		}
	})

	t.Run("with an error and a handshake", func(t *testing.T) {
		handshake := &model.ArchivalTLSOrQUICHandshakeResult{Address: "1.1.1.1:443"}
		iter := newIterationFromHandshake(3, errors.New("mocked error"), nil, handshake, "example.com")
		if iter.Handshake != handshake {
			t.Fatal("expected the recorded handshake to be kept")
		}
		if iter.Handshake.Failure == nil || iter.Handshake.ServerName != "example.com" {
			t.Fatal("expected the failure and servername to be set")
		}
	})
}

func TestHandshakeWithTTLDialFailure(t *testing.T) {
	// obtain an address where nothing is listening
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := listener.Addr().String()
	listener.Close()

	m := NewExperimentMeasurer()
	tr := &IterativeTrace{}
	wg := new(sync.WaitGroup)
	wg.Add(1)
	m.handshakeWithTTL(context.Background(), 0, time.Now(), model.DiscardLogger, address, "example.com", 3, tr, wg, &Config{})
	if len(tr.Iterations) != 1 {
		t.Fatal("unexpected number of iterations")
	}
	iter := tr.Iterations[0]
	if iter.Handshake == nil || iter.Handshake.Failure == nil {
		t.Fatal("expected a dial failure")
	}
}
