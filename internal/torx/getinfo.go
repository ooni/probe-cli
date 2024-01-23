package torx

//
// getinfo.go - implements the GETINFO command.
//
// SPDX-License-Identifier: MIT
//
// Adapted from https://github.com/cretz/bine.
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/optional"
	"github.com/ooni/probe-cli/v3/internal/torcontrolnet"
)

// GetInfo sends the GETINFO command.
func GetInfo(ctx context.Context, conn ControlTransport, key string) ([]*KeyValuePair, error) {
	// send request and receive the response
	resp, err := conn.SendRecv(ctx, "GETINFO %s", key)
	if err != nil {
		return nil, err
	}

	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return nil, ErrControlRequestFailed
	}

	// For each line returned by tor the format is either
	//
	//	250-keyword=value
	//
	// or
	//
	//	250+keyword=
	//	value
	//	.
	//
	// However, note that the readloop will transform the latter
	// to keyword=value, where value contains >= 0 newlines.
	var result []*KeyValuePair
	for _, entry := range resp.Data {
		key, value, ok := utilsPartitionString(entry, '=')
		if !ok {
			// we need to have the equal sign
			continue
		}
		value, err := utilsUnescapeSimpleQuotedStringIfNeeded(value)
		if err != nil {
			// something is wrong with this value
			continue
		}
		pair := &KeyValuePair{
			Key:   key,
			Value: optional.Some(value),
		}
		result = append(result, pair)
	}

	return result, nil
}
