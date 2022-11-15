package openvpn

import (
	"time"

	"github.com/ooni/minivpn/vpn"
)

// HandshakeEvent captures a uint16 event measuring the progress of the
// handshake for the OpenVPN connection.
type HandshakeEvent struct {
	TransactionID uint16  `json:"transaction_id"`
	Operation     string  `json:"operation"`
	Time          float64 `json:"t"`
}

func newHandshakeEvent(evt uint16, t time.Duration) HandshakeEvent {
	var s string
	switch evt {
	case vpn.EventReady:
		s = "ready"
	case vpn.EventDialDone:
		s = "dial_done"
	case vpn.EventHandshake:
		s = "vpn_handshake_start"
	case vpn.EventReset:
		s = "reset"
	case vpn.EventTLSConn:
		s = "tls_conn"
	case vpn.EventTLSHandshake:
		s = "tls_handshake_start"
	case vpn.EventTLSHandshakeDone:
		s = "tls_handshake_done"
	case vpn.EventDataInitDone:
		s = "data_init"
	case vpn.EventHandshakeDone:
		s = "vpn_handshake_done"
	default:
		s = "unknown"
	}
	return HandshakeEvent{
		TransactionID: evt,
		Operation:     s,
		Time:          toMs(t),
	}
}
