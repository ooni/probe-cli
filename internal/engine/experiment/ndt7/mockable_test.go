package ndt7

import (
	"io"
	"time"

	"github.com/gorilla/websocket"
)

type mockableConnMock struct {
	NextReaderMsgType       int
	NextReaderErr           error
	NextReaderReader        func() io.Reader
	ReadDeadlineErr         error
	WriteDeadlineErr        error
	WritePreparedMessageErr error
}

func (c *mockableConnMock) NextReader() (int, io.Reader, error) {
	var reader io.Reader
	if c.NextReaderReader != nil {
		reader = c.NextReaderReader()
	}
	return c.NextReaderMsgType, reader, c.NextReaderErr
}

func (c *mockableConnMock) SetReadDeadline(time.Time) error {
	return c.ReadDeadlineErr
}

func (c *mockableConnMock) SetReadLimit(int64) {}

func (c *mockableConnMock) SetWriteDeadline(time.Time) error {
	return c.WriteDeadlineErr
}

func (c *mockableConnMock) WritePreparedMessage(*websocket.PreparedMessage) error {
	return c.WritePreparedMessageErr
}
