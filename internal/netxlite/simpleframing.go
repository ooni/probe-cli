package netxlite

import (
	"errors"
	"io"
	"math"
)

// MaxSimpleFrameSize is the maximum size of a simple frame.
const MaxSimpleFrameSize = math.MaxUint16

// ErrSimpleFrameSize indicates that a frame is too large to be sent.
var ErrSimpleFrameSize = errors.New("netxlite: frame larger than 2^16")

// SendSimpleFrame sends a simple frame over the given io.Writer.
func WriteSimpleFrame(conn io.Writer, frame []byte) error {
	if len(frame) > MaxSimpleFrameSize {
		return ErrSimpleFrameSize
	}
	buf := []byte{byte(len(frame) >> 8)}
	buf = append(buf, byte(len(frame)))
	buf = append(buf, frame...)
	_, err := conn.Write(buf)
	return err
}

// RecvSimpleFrame receives a simple frame from the given io.Reader.
func ReadSimpleFrame(conn io.Reader) ([]byte, error) {
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(header[0])<<8 | int(header[1])
	frame := make([]byte, length)
	if _, err := io.ReadFull(conn, frame); err != nil {
		return nil, err
	}
	return frame, nil
}
