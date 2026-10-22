package tlsmiddlebox

import (
	"strings"
	"testing"
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
