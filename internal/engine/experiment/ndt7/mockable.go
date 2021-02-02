package ndt7

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type mockableConn interface {
	NextReader() (int, io.Reader, error)
	SetReadDeadline(time.Time) error
	SetReadLimit(int64)
	SetWriteDeadline(time.Time) error
	WritePreparedMessage(*websocket.PreparedMessage) error
}
