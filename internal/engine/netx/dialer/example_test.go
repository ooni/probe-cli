package dialer_test

import (
	"context"
	"net"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

func Example() {
	saver := &trace.Saver{}

	dlr := dialer.New(&dialer.Config{
		DialSaver:      saver,
		Logger:         log.Log,
		ReadWriteSaver: saver,
	}, &net.Resolver{})

	ctx := context.Background()
	conn, err := dlr.DialContext(ctx, "tcp", "8.8.8.8:53")
	if err != nil {
		log.WithError(err).Fatal("DialContext failed")
	}

	// ... use the connection ...

	conn.Close()
}
