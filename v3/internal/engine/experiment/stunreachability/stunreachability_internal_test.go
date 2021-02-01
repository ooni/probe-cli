package stunreachability

import (
	"context"
	"net"

	"github.com/pion/stun"
)

func (c *Config) SetNewClient(
	f func(conn stun.Connection, options ...stun.ClientOption) (*stun.Client, error)) {
	c.newClient = f
}

func (c *Config) SetDialContext(
	f func(ctx context.Context, network, address string) (net.Conn, error)) {
	c.dialContext = f
}
