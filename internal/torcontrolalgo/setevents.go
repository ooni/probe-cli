package torcontrolalgo

//
// setevents.go - implements the SETEVENTS command.
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

// Event codes
const (
	EventAddrMap           = "ADDRMAP"
	EventBandwidth         = "BW"
	EventBuildTimeoutSet   = "BUILDTIMEOUT_SET"
	EventCellStats         = "CELL_STATS"
	EventCircuit           = "CIRC"
	EventCircuitBandwidth  = "CIRC_BW"
	EventCircuitMinor      = "CIRC_MINOR"
	EventClientsSeen       = "CLIENTS_SEEN"
	EventConfChanged       = "CONF_CHANGED"
	EventConnBandwidth     = "CONN_BW"
	EventDescChanged       = "DESCCHANGED"
	EventGuard             = "GUARD"
	EventHSDesc            = "HS_DESC"
	EventHSDescContent     = "HS_DESC_CONTENT"
	EventLogDebug          = "DEBUG"
	EventLogErr            = "ERR"
	EventLogInfo           = "INFO"
	EventLogNotice         = "NOTICE"
	EventLogWarn           = "WARN"
	EventNetworkLiveness   = "NETWORK_LIVENESS"
	EventNetworkStatus     = "NS"
	EventNewConsensus      = "NEWCONSENSUS"
	EventNewDesc           = "NEWDESC"
	EventORConn            = "ORCONN"
	EventSignal            = "SIGNAL"
	EventStatusClient      = "STATUS_CLIENT"
	EventStatusGeneral     = "STATUS_GENERAL"
	EventStatusServer      = "STATUS_SERVER"
	EventStream            = "STREAM"
	EventStreamBandwidth   = "STREAM_BW"
	EventTokenBucketEmpty  = "TB_EMPTY"
	EventTransportLaunched = "TRANSPORT_LAUNCHED"
)

// SetEvents sends the SETEVENTS command.
func SetEvents(ctx context.Context, conn Conn, values ...string) error {
	// send request and receive the response
	resp, err := conn.SendRecv(ctx, "SETEVENTS %s", strings.Join(values, " "))
	if err != nil {
		return err
	}

	// make sure the response is successful
	if resp.Status != torcontrolnet.StatusOk {
		return ErrRequestFailed
	}
	return nil
}
