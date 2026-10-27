//go:build !linux

package quicfp

import "github.com/ooni/probe-cli/v3/internal/model"

// disablePathMTUDiscovery is a no-op on non-Linux platforms. The
// ip_fragmentation vector still sends an oversized datagram either way;
// whether that results in genuine local IP fragmentation, an ICMP error,
// or something else depends entirely on the OS/network stack's own
// defaults, which this codebase does not attempt to override outside
// Linux.
func disablePathMTUDiscovery(conn model.UDPLikeConn) {}
