package webconnectivity

//
// TCP/IP analysis
//

import (
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// analysisTCPIPToplevel is the toplevel analysis function for TCP/IP results.
func (tk *TestKeys) analysisTCPIPToplevel(logger model.Logger) {
	// if we don't have a control result, do nothing.
	if tk.Control == nil || len(tk.Control.TCPConnect) <= 0 {
		return
	}
	var (
		istrue  = true
		isfalse = false
	)

	// walk the list of probe results and compare with TH results
	for _, entry := range tk.TCPConnect {
		// skip successful entries
		failure := entry.Status.Failure
		if failure == nil {
			entry.Status.Blocked = &isfalse
			continue // did not fail
		}
		// make sure we exclude the IPv6 failures caused by lack of
		// proper IPv6 support by the probe
		ipv6, err := netxlite.IsIPv6(entry.IP)
		if err != nil {
			continue // looks like a bug
		}
		if ipv6 {
			ignore := (*failure == netxlite.FailureNetworkUnreachable ||
				*failure == netxlite.FailureHostUnreachable)
			if ignore {
				// this occurs when we don't have IPv6 on the probe
				continue
			}
		}
		// obtain the corresponding endpoint
		epnt := net.JoinHostPort(entry.IP, fmt.Sprintf("%d", entry.Port))
		ctrl, found := tk.Control.TCPConnect[epnt]
		if !found {
			continue // only the probe tested this, so hard to say anything
		}
		if ctrl.Failure != nil {
			// if the TH failed as well, don't set any blocking flag
			entry.Status.Blocked = &isfalse
			continue
		}
		logger.Warnf("TCP/IP: endpoint %s is blocked (see #%d)", epnt, entry.TransactionID)
		entry.Status.Blocked = &istrue
		tk.BlockingFlags |= analysisBlockingTCPIP
	}
}
