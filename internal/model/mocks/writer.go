package mocks

import "io"

// Writer allows to mock any io.Writer.
type Writer struct {
	MockWrite func(b []byte) (int, error)
}

// Writer implements an io.Writer.
var _ io.Writer = &Writer{}

// Write implements io.Writer.Write.
func (r *Writer) Write(b []byte) (int, error) {
	return r.MockWrite(b)
}
