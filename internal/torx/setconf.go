package torx

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

	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// SetConf sends the SETCONF command.
func SetConf(ctx context.Context, conn ControlTransport, values ...*KeyValuePair) error {
	// prepare a single string containing all the config key=value pairs
	var entries []string
	for _, value := range values {
		entries = append(entries, value.Key)
		if !value.Value.IsNone() {
			entries = append(entries, "=")
			entries = append(entries, utilsEscapeSimpleQuotedStringIfNeeded(value.Value.Unwrap()))
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
		return ErrControlRequestFailed
	}
	return nil
}
