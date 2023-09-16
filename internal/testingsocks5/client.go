package testingsocks5

import (
	"errors"
	"fmt"
	"io"
	"net"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// client is a minimal client used for testing the server
type client struct {
	exchanges []exchange
}

type exchange struct {
	send   []byte
	expect []byte
}

var errUnexpectedResponse = errors.New("unexpected response")

func (ic *client) run(logger model.Logger, conn net.Conn) error {
	for _, exchange := range ic.exchanges {
		logger.Infof("SOCKS5_CLIENT: sending: %v", exchange.send)
		if _, err := conn.Write(exchange.send); err != nil {
			return err
		}
		logger.Infof("SOCKS5_CLIENT: expecting: %v", exchange.expect)
		buffer := make([]byte, len(exchange.expect))
		if _, err := io.ReadFull(conn, buffer); err != nil {
			return err
		}
		logger.Infof("SOCKS5_CLIENT: got: %v", buffer)
		if diff := cmp.Diff(exchange.expect, buffer); diff != "" {
			return fmt.Errorf("%w: %s", errUnexpectedResponse, diff)
		}
	}
	return nil
}
