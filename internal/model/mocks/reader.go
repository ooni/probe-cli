package mocks

import "io"

// Reader allows to mock any io.Reader.
type Reader struct {
	MockRead func(b []byte) (int, error)
}

// MockableReader implements an io.Reader.
var _ io.Reader = &Reader{}

// Read implements io.Reader.Read.
func (r *Reader) Read(b []byte) (int, error) {
	return r.MockRead(b)
}
