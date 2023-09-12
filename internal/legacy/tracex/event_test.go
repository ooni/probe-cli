package tracex

import "testing"

func TestUnusedEventsNames(t *testing.T) {
	// Tests that we don't break the names of events we're currently
	// not getting the name of directly even if they're saved.

	t.Run("EventQUICHandshakeStart", func(t *testing.T) {
		ev := &EventQUICHandshakeStart{}
		if ev.Name() != "quic_handshake_start" {
			t.Fatal("invalid event name")
		}
	})

	t.Run("EventQUICHandshakeDone", func(t *testing.T) {
		ev := &EventQUICHandshakeDone{}
		if ev.Name() != "quic_handshake_done" {
			t.Fatal("invalid event name")
		}
	})

	t.Run("EventTLSHandshakeStart", func(t *testing.T) {
		ev := &EventTLSHandshakeStart{}
		if ev.Name() != "tls_handshake_start" {
			t.Fatal("invalid event name")
		}
	})

	t.Run("EventTLSHandshakeDone", func(t *testing.T) {
		ev := &EventTLSHandshakeDone{}
		if ev.Name() != "tls_handshake_done" {
			t.Fatal("invalid event name")
		}
	})
}
