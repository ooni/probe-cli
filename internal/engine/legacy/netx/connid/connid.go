// Package connid contains code to generate the connectionID
package connid

import (
	"net"
	"strconv"
	"strings"
)

// Compute computes the connectionID from the local socket address. The zero
// value is conventionally returned to mean "unknown".
func Compute(network, address string) int64 {
	_, portstring, err := net.SplitHostPort(address)
	if err != nil {
		return 0
	}
	portnum, err := strconv.Atoi(portstring)
	if err != nil {
		return 0
	}
	if portnum < 0 || portnum > 65535 {
		return 0
	}
	result := int64(portnum)
	if strings.Contains(network, "udp") {
		result *= -1
	} else if !strings.Contains(network, "tcp") {
		result = 0
	}
	return result
}
