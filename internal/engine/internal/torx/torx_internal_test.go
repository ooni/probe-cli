package torx

import (
	"net/url"
	"time"
)

func NewTunnel(bootstrapTime time.Duration, instance TorProcess, proxy *url.URL) *Tunnel {
	return &Tunnel{
		bootstrapTime: bootstrapTime,
		instance:      instance,
		proxy:         proxy,
	}
}
