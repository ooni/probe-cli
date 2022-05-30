package ndt7

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

// weConn is the interface of gorilla/websocket.Conn
type wsConn interface {
	NextReader() (int, io.Reader, error)
	SetReadDeadline(time.Time) error
	SetReadLimit(int64)
	SetWriteDeadline(time.Time) error
	WritePreparedMessage(*websocket.PreparedMessage) error
}
