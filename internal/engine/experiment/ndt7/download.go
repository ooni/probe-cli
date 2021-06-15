package ndt7

import (
	"context"
	"errors"
	"io"
	"time"

	"github.com/gorilla/websocket"
	"github.com/ooni/probe-cli/v3/internal/iox"
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
	return mgr.reduceErr(mgr.doRun(ctx))
}

// reduceErr treats as non-errors the errors caused by the context
// so that we can focus instead on network errors.
//
// This function was introduced by https://github.com/ooni/probe-cli/pull/379
// since before such a PR we did not see context interrupting
// errors when we were reading messages. Since before such a PR
// we used to return `nil` on context errors, this function is
// here to keep the previous behavior by filtering the error
// returned when reading messages, given that now reading messages
// can fail midway because we use iox.ReadAllContext.
func (mgr downloadManager) reduceErr(err error) error {
	if errors.Is(err, context.Canceled) {
		return nil
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return nil
	}
	return err
}

func (mgr downloadManager) doRun(ctx context.Context) error {
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
			data, err := iox.ReadAllContext(ctx, reader)
			if err != nil {
				return err
			}
			total += int64(len(data))
			if err := mgr.onJSON(data); err != nil {
				return err
			}
			continue
		}
		n, err := iox.CopyContext(ctx, io.Discard, reader)
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
