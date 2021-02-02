package ndt7

import (
	"context"
	"time"

	"github.com/gorilla/websocket"
)

func newMessage(n int) (*websocket.PreparedMessage, error) {
	return websocket.NewPreparedMessage(websocket.BinaryMessage, make([]byte, n))
}

type uploadManager struct {
	conn                 mockableConn
	fractionForScaling   int64
	maxRuntime           time.Duration
	maxMessageSize       int
	maxScaledMessageSize int
	measureInterval      time.Duration
	minMessageSize       int
	newMessage           func(int) (*websocket.PreparedMessage, error)
	onPerformance        callbackPerformance
}

func newUploadManager(
	conn mockableConn, onPerformance callbackPerformance,
) uploadManager {
	return uploadManager{
		conn:                 conn,
		fractionForScaling:   paramFractionForScaling,
		maxRuntime:           paramMaxRuntime,
		maxMessageSize:       paramMaxMessageSize,
		maxScaledMessageSize: paramMaxScaledMessageSize,
		measureInterval:      paramMeasureInterval,
		minMessageSize:       paramMinMessageSize,
		newMessage:           newMessage,
		onPerformance:        onPerformance,
	}
}

func (mgr uploadManager) run(ctx context.Context) error {
	var total int64
	start := time.Now()
	if err := mgr.conn.SetWriteDeadline(time.Now().Add(mgr.maxRuntime)); err != nil {
		return err
	}
	size := mgr.minMessageSize
	message, err := mgr.newMessage(size)
	if err != nil {
		return err
	}
	ticker := time.NewTicker(mgr.measureInterval)
	defer ticker.Stop()
	for ctx.Err() == nil {
		if err := mgr.conn.WritePreparedMessage(message); err != nil {
			return err
		}
		total += int64(size)
		select {
		case now := <-ticker.C:
			mgr.onPerformance(now.Sub(start), total)
		default:
			// NOTHING
		}
		if size >= mgr.maxScaledMessageSize || int64(size) >= (total/mgr.fractionForScaling) {
			continue
		}
		size <<= 1
		if message, err = mgr.newMessage(size); err != nil {
			return err
		}
	}
	return nil
}
