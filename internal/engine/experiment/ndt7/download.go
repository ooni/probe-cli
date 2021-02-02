package ndt7

import (
	"context"
	"io"
	"io/ioutil"
	"time"

	"github.com/gorilla/websocket"
)

type downloadManager struct {
	conn            mockableConn
	maxMessageSize  int64
	maxRuntime      time.Duration
	measureInterval time.Duration
	onJSON          callbackJSON
	onPerformance   callbackPerformance
}

func newDownloadManager(
	conn mockableConn, onPerformance callbackPerformance,
	onJSON callbackJSON,
) downloadManager {
	return downloadManager{
		conn:            conn,
		maxMessageSize:  paramMaxMessageSize,
		maxRuntime:      paramMaxRuntime,
		measureInterval: paramMeasureInterval,
		onJSON:          onJSON,
		onPerformance:   onPerformance,
	}
}

func (mgr downloadManager) run(ctx context.Context) error {
	var total int64
	start := time.Now()
	if err := mgr.conn.SetReadDeadline(start.Add(mgr.maxRuntime)); err != nil {
		return err
	}
	mgr.conn.SetReadLimit(mgr.maxMessageSize)
	ticker := time.NewTicker(mgr.measureInterval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		kind, reader, err := mgr.conn.NextReader()
		if err != nil {
			return err
		}
		if kind == websocket.TextMessage {
			data, err := ioutil.ReadAll(reader)
			if err != nil {
				return err
			}
			total += int64(len(data))
			if err := mgr.onJSON(data); err != nil {
				return err
			}
			continue
		}
		n, err := io.Copy(ioutil.Discard, reader)
		if err != nil {
			return err
		}
		total += int64(n)
		select {
		case now := <-ticker.C:
			mgr.onPerformance(now.Sub(start), total)
		default:
			// NOTHING
		}
	}
	return nil
}
