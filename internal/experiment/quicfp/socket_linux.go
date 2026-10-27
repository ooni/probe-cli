//go:build linux

package quicfp

import (
	"net"
	"syscall"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Linux's linux/in.h defines these; they're inlined here (rather than
// imported from golang.org/x/sys/unix) to keep this package
// dependency-free.
const (
	ipMTUDiscover  = 10 // IP_MTU_DISCOVER
	ipPMTUDiscDont = 0  // IP_PMTUDISC_DONT
)

// disablePathMTUDiscovery best-effort configures conn so the kernel
// fragments oversized outgoing UDP datagrams locally instead of relying on
// Path MTU Discovery (Linux's default), which would otherwise drop an
// over-MTU datagram and return an ICMP "fragmentation needed" error instead
// of ever putting it on the wire. Failures are ignored: not every
// kernel/network namespace allows changing this, and callers should treat
// the ip_fragmentation vector's results as best-effort regardless.
func disablePathMTUDiscovery(conn model.UDPLikeConn) {
	uc, ok := conn.(*net.UDPConn)
	if !ok {
		return
	}
	rawConn, err := uc.SyscallConn()
	if err != nil {
		return
	}
	_ = rawConn.Control(func(fd uintptr) {
		_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_IP, ipMTUDiscover, ipPMTUDiscDont)
	})
}
