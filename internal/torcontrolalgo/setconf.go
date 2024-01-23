package torcontrolalgo

//
// setconf.go - implements the SETCONF command.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// SetConfDisableNetwork sends the SETCONF DisableNetwork 1 command.
func SetConfDisableNetwork(ctx context.Context, conn Conn) error {
	return setConfDisableNetwork(ctx, conn, "1")
}

// SetConfEnableNetwork sends the SETCONF DisableNetwork 0 command.
func SetConfEnableNetwork(ctx context.Context, conn Conn) error {
	return setConfDisableNetwork(ctx, conn, "0")
}

func setConfDisableNetwork(ctx context.Context, conn Conn, value string) error {
	return SetConf(ctx, conn, NewKeyValuePair("DisableNetwork", optional.Some(value)))
}

// SetConf sends the SETCONF command.
func SetConf(ctx context.Context, conn Conn, values ...*KeyValuePair) error {
	// prepare a single string containing all the config key=value pairs
	var entries []string
	for _, value := range values {
		entries = append(entries, value.Key)
		if !value.Value.IsNone() {
			entries = append(entries, "=")
			entries = append(entries, escapeSimpleQuotedStringIfNeeded(value.Value.Unwrap()))
		}
		entries = append(entries, " ")
	}

	// send request and receive the response
	resp, err := conn.SendRecv(ctx, "SETCONF %s", strings.Join(entries, ""))
	if err != nil {
		return err
	}

	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return ErrRequestFailed
	}
	return nil
}
