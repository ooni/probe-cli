package wireguard

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

type SpeedTest struct {
	IsVPN      bool    `json:"is_vpn"`
	Failed     bool    `json:"failed"`
	Failure    *string `json:"failure"`
	File       string  `json:"file"`
	T0         float64 `json:"t0"`
	T          float64 `json:"t"`
	BodyLength int64   `json:"x_body_length"`
}

// TestKeys contains the experiment's result.
type TestKeys struct {
	//
	// Keys that will serve as primary keys.
	//

	// Provider is the entity that controls the endpoints.
	Provider string `json:"provider"`

	// Proto is the protocol used in the experiment (wireguard in this case).
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

	// SuccessICMP signals an experiment in which _all_ of the first two ICMP pings
	// have less than 50% packet loss.
	SuccessICMP bool `json:"success_icmp"`

	// SuccessURLGrab signals an experiment in which at least one of the urlgrabs through the tunnel is successful.
	SuccessURLGrab bool `json:"success_urlgrab"`

	// Success is true when we reached the end of the experiment without errors.
	Success bool `json:"success"`

	// Failure contains the failure string or nil.
	Failure *string `json:"failure"`

	// Pings is an array of ping stats.
	Pings []*PingResult `json:"icmp_pings"`

	SpeedTest []*SpeedTest `json:"speed_test"`

	// Requests contain HTTP results done through the tunnel.
	Requests []model.ArchivalHTTPRequestResult `json:"requests"`
}
