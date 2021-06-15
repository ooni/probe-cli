// Package iox contains io extensions.
package iox

import (
	"context"
	"io"
)

// ReadAllContext reads the whole reader r in a
// background goroutine. This function will return
// earlier if the context is cancelled. In which case
// we will continue reading from r in the background
// goroutine, and we will discard the result. To stop
// the long-running goroutine, you need to close the
// connection bound to the r reader, if possible.
func ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error) {
	datach, errch := make(chan []byte, 1), make(chan error, 1) // buffers
	go func() {
		data, err := io.ReadAll(r)
		if err != nil {
			errch <- err
			return
		}
		datach <- data
	}()
	select {
	case data := <-datach:
		return data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case err := <-errch:
		return nil, err
	}
}

// MockableReader allows to mock any io.Reader.
type MockableReader struct {
	MockRead func(b []byte) (int, error)
}

// MockableReader implements an io.Reader.
var _ io.Reader = &MockableReader{}

// Read implements io.Reader.Read.
func (r *MockableReader) Read(b []byte) (int, error) {
	return r.MockRead(b)
}
