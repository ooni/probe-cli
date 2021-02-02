package resolver

import (
	"context"
	"time"
)

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}

// LoggingResolver is a resolver that emits events
type LoggingResolver struct {
	Resolver
	Logger Logger
}

// LookupHost returns the IP addresses of a host
func (r LoggingResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	r.Logger.Debugf("resolve %s...", hostname)
	start := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	stop := time.Now()
	r.Logger.Debugf("resolve %s... (%+v, %+v) in %s", hostname, addrs, err, stop.Sub(start))
	return addrs, err
}

var _ Resolver = LoggingResolver{}
