package netplumbing

// This file contains the implementation of ReadAllContext.

import (
	"context"
	"io"
	"io/ioutil"
)

// ReadAllContext reads all the bytes from the given reader. If the
// context is interrupted while the read is in progress, this function
// will return early and ignore the result of reading. In such case,
// you typically want to close the response body from which you're
// reading so that you also interrupt the background goroutine which
// is reading data from the input reader.
//
// Using this function is the recommended way to guarantee that you
// are not going to block for $censorship when reading bodies.
func ReadAllContext(ctx context.Context, r io.Reader) ([]byte, error) {
	outch, errch := make(chan []byte, 1), make(chan error, 1)
	go func() {
		data, err := ioutil.ReadAll(r)
		if err != nil {
			errch <- err
			return
		}
		outch <- data
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-outch:
		return out, nil
	case err := <-errch:
		return nil, err
	}
}
