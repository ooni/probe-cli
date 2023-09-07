package remote

import (
	"errors"
	"io"
)

// ReadPacket reads an IP packet prefixed by a three-bytes length from a reader.
func ReadPacket(reader io.Reader) ([]byte, error) {
	header := make([]byte, 3)
	if _, err := io.ReadFull(reader, header); err != nil {
		return nil, err
	}
	var length int
	length |= int(header[0]) << 16
	length |= int(header[1]) << 8
	length |= int(header[2]) << 0
	ipPacket := make([]byte, length)
	if _, err := io.ReadFull(reader, ipPacket); err != nil {
		return nil, err
	}
	return ipPacket, nil
}

// MaxPacketSize is the maximum packet size.
const MaxPacketSize = (1 << 24) - 1

// ErrPacketTooBig indicates that a packet is too big
var ErrPacketTooBig = errors.New("remote: packet too big")

// WritePacket writes an IP packet prefixed by a three-bytes length to a writer.
func WritePacket(writer io.Writer, ipPacket []byte) error {
	length := len(ipPacket)
	if length > MaxPacketSize {
		return ErrPacketTooBig
	}
	data := make([]byte, 3)
	data[0] = byte((length >> 16) & 0xff)
	data[1] = byte((length >> 8) & 0xff)
	data[2] = byte((length >> 0) & 0xff)
	data = append(data, ipPacket...)
	_, err := writer.Write(data)
	return err
}
