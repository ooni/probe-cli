package main

import (
	"net"
	"sync"
)

// tcpState contains state for TCP DNAT.
type tcpState struct {
	// dnat maps state keys to the source address
	dnat map[uint16]net.IP

	// mu provides mutual exclusion
	mu sync.Mutex
}
