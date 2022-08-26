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
//
// This algorithm has two objectives:
//
// 1. walk the list of TCP connect attempts and mark each of them as
// Status.Blocked = true | false | null depending on what the TH observed
// for the same set of IP addresses (it's ugly to modify a data struct
// in place, but this algorithm is defined by the spec);
//
// 2. assign the analysisFlagTCPIPBlocking flag to XBlockingFlags if
// we see any TCP endpoint for which Status.Blocked is true.
func (tk *TestKeys) analysisTCPIPToplevel(logger model.Logger) {
	// if we don't have a control result, do nothing.
	if tk.Control == nil || len(tk.Control.TCPConnect) <= 0 {
		return
	}
	var (
		istrue  = true
		isfalse = false
	)

	// TODO(bassosimone): the TH should measure also some of the IP addrs it discovered
	// and the probe did not discover to improve the analysis. Otherwise, the probe
	// is fooled by the TH also failing for countries that return random IP addresses
	// that are actually not working. Yet, ooni/data would definitely see this.

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
			continue // only the probe tested this, so hard to say anything...
		}
		if ctrl.Failure != nil {
			// If the TH failed as well, don't set XBlockingFlags and
			// also don't bother with setting .Status.Blocked thus leaving
			// it null. Performing precise error mapping should be a job
			// for the pipeline rather than for the probe.
			continue
		}
		logger.Warnf("TCP/IP: endpoint %s is blocked (see #%d)", epnt, entry.TransactionID)
		entry.Status.Blocked = &istrue
		tk.BlockingFlags |= analysisFlagTCPIPBlocking
	}
}
