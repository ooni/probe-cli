package openvpn

// TODO(ainghazal): move to archiva package when consolidated.

// ArchivalOpenVPNHandshakeResult contains the result of a OpenVPN handshake.
type ArchivalOpenVPNHandshakeResult struct {
	BootstrapTime float64                      `json:"bootstrap_time,omitempty"`
	Endpoint      string                       `json:"endpoint"`
	IP            string                       `json:"ip"`
	Port          int                          `json:"port"`
	Provider      string                       `json:"provider"`
	Status        ArchivalOpenVPNConnectStatus `json:"status"`
	T0            float64                      `json:"t0,omitempty"`
	T             float64                      `json:"t"`
	Tags          []string                     `json:"tags"`
	TransactionID int64                        `json:"transaction_id,omitempty"`
}

// ArchivalOpenVPNConnectStatus is the status of ArchivalOpenVPNConnectResult.
type ArchivalOpenVPNConnectStatus struct {
	Blocked *bool   `json:"blocked,omitempty"`
	Failure *string `json:"failure"`
	Success bool    `json:"success"`
}

type ArchivalNetworkEvent struct {
	// TODO(ainghazal): need to properly propagate I/O errors during the handshake.
	Address       string   `json:"address,omitempty"`
	Failure       *string  `json:"failure"`
	NumBytes      int64    `json:"num_bytes,omitempty"`
	Operation     string   `json:"operation"`
	Proto         string   `json:"proto,omitempty"`
	T0            float64  `json:"t0,omitempty"`
	T             float64  `json:"t"`
	TransactionID int64    `json:"transaction_id,omitempty"`
	Tags          []string `json:"tags,omitempty"`
}