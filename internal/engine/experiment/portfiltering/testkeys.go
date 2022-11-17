package portfiltering

import "github.com/ooni/probe-cli/v3/internal/model"

// TestKeys contains the experiment results.
type TestKeys struct {
	TCPConnect *model.ArchivalTCPConnectResult `json:"tcp_connect"`
}
