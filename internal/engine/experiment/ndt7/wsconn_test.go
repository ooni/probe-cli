package ndt7

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type mockableWSConn struct {
	NextReaderMsgType       int
	NextReaderErr           error
	NextReaderReader        func() io.Reader
	ReadDeadlineErr         error
	WriteDeadlineErr        error
	WritePreparedMessageErr error
}

func (c *mockableWSConn) NextReader() (int, io.Reader, error) {
	var reader io.Reader
	if c.NextReaderReader != nil {
		reader = c.NextReaderReader()
	}
	return c.NextReaderMsgType, reader, c.NextReaderErr
}

func (c *mockableWSConn) SetReadDeadline(time.Time) error {
	return c.ReadDeadlineErr
}

func (c *mockableWSConn) SetReadLimit(int64) {}

func (c *mockableWSConn) SetWriteDeadline(time.Time) error {
	return c.WriteDeadlineErr
}

func (c *mockableWSConn) WritePreparedMessage(*websocket.PreparedMessage) error {
	return c.WritePreparedMessageErr
}
