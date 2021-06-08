package dialer

import (
	"context"
	"net"
	"time"
)

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}

// LoggingDialer is a Dialer with logging
type LoggingDialer struct {
	Dialer
	Logger Logger
}

// DialContext implements Dialer.DialContext
func (d LoggingDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.Logger.Debugf("dial %s/%s...", address, network)
	start := time.Now()
	conn, err := d.Dialer.DialContext(ctx, network, address)
	stop := time.Now()
	d.Logger.Debugf("dial %s/%s... %+v in %s", address, network, err, stop.Sub(start))
	return conn, err
}
