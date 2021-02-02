package internal

import (
	"net"
	"time"
)

// SleeperWriter is a net.Conn that optionally sleeps for the
// specified delay before posting each write.
type SleeperWriter struct {
	net.Conn
	Delay time.Duration
}

func (c SleeperWriter) Write(b []byte) (int, error) {
	<-time.After(c.Delay)
	return c.Conn.Write(b)
}

// SplitterWriter is a writer that splits every outgoing buffer
// according to the rules specified by the Splitter.
//
// Caveat
//
// The TLS ClientHello may be retransmitted if the server is
// requesting us to restart the negotiation. Therefore, it is
// not safe to just run the splitting once. Since this code
// is meant to investigate TLS blocking, that's fine.
type SplitterWriter struct {
	net.Conn
	Splitter func([]byte) [][]byte
}

// Write implements net.Conn.Write
func (c SplitterWriter) Write(b []byte) (int, error) {
	if c.Splitter != nil {
		return Writev(c.Conn, c.Splitter(b))
	}
	return c.Conn.Write(b)
}

// Writev writes all the vectors inside datalist using the specified
// conn. Returns either an error or the number of bytes sent. Note
// that this function skips any empty entry in datalist.
func Writev(conn net.Conn, datalist [][]byte) (int, error) {
	var total int
	for _, data := range datalist {
		if len(data) > 0 {
			count, err := conn.Write(data)
			if err != nil {
				return 0, err
			}
			total += count
		}
	}
	return total, nil
}
