package openvpn

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TestKeys contains the experiment's result.
type TestKeys struct {
	//
	// Keys that will serve as primary keys.
	//

	// Provider is the entity that controls the endpoints.
	Provider string `json:"provider"`

	// Proto is the protocol used in the experiment (openvpn in this case).
	Proto string `json:"vpn_protocol"`

	// Transport is the transport protocol (tcp, udp).
	Transport string `json:"transport"`

	// Remote is the remote used in the experiment (ip:addr).
	Remote string `json:"remote"`

	// Obfuscation is the kind of obfuscation used, if any.
	Obfuscation string `json:"obfuscation"`

	//
	// Other keys
	//

	// BootstrapTime contains the bootstrap time on success.
	BootstrapTime float64 `json:"bootstrap_time"`

	// HandshakeEvents is a sequence of handshake events with their corresponding timestamp.
	HandshakeEvents []HandshakeEvent `json:"network_events"`

	// Last known received OpenVPN handshake event
	LastHandshakeTransactionID int `json:"last_handshake_transaction_id"`

	// TCPCconnect traces a TCP connection for the vpn dialer (null for UDP transport).
	TCPConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// Pings holds an array for aggregated stats of each ping.
	Pings []*PingResult `json:"icmp_pings"`

	// Requests contain HTTP results done through the tunnel.
	Requests []model.ArchivalHTTPRequestResult `json:"requests"`

	// Software identification

	// MiniVPNVersion contains the version of the minivpn library used.
	MiniVPNVersion string `json:"minivpn_version"`

	// Obfs4Version contains the version of the obfs4 library used.
	Obfs4Version string `json:"obfs4_version"`

	// Summaries for partial results

	// SuccessHandshake is true when we reach the last handshake stage.
	SuccessHandshake bool `json:"success_handshake"`

	// SuccessICMP signals an experiment in which _all_ of the first two ICMP pings
	// have less than 50% packet loss.
	SuccessICMP bool `json:"success_icmp"`

	// SuccessURLGrab signals an experiment in which at least one of the urlgrabs through the tunnel is successful.
	SuccessURLGrab bool `json:"success_urlgrab"`

	// Success is true when we reached the end of the test without errors.
	Success bool `json:"success"`
}

// NewTestKeys creates a new instance of TestKeys.
func NewTestKeys() *TestKeys {
	tk := &TestKeys{
		Proto:           testName,
		MiniVPNVersion:  getMiniVPNVersion(),
		Obfs4Version:    getObfs4Version(),
		BootstrapTime:   0,
		Failure:         nil,
		HandshakeEvents: []HandshakeEvent{},
	}
	return tk
}
