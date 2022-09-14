package webconnectivity

//
// Extensions to incrementally stream-reading a response body.
//

import (
	"context"
	"errors"
	"io"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// StreamAllContext streams from the given reader [r] until
// interrupted by [ctx] or when [r] hits the EOF.
//
// This function runs a background goroutine that should exit as soon
// as [ctx] is done or when [reader] is closed, if applicable.
//
// This function transforms an errors.Is(err, io.EOF) to a nil error
// such as the standard library's ReadAll does.
//
// This function might return a non-zero-length buffer along with
// an non-nil error in the case in which we could only read a portion
// of the body and then we were interrupted by the error.
func StreamAllContext(ctx context.Context, reader io.Reader) ([]byte, error) {
	// TODO(bassosimone): consider merging into the ./internal/netxlite/iox.go file
	// once this code has been used in testing for quite some time
	datach, errch := make(chan []byte), make(chan error)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go func() {
		buffer := make([]byte, 1<<13)
		for {
			count, err := reader.Read(buffer)
			if count > 0 {
				data := buffer[:count]
				select {
				case datach <- data:
					// fallthrough to check error
				case <-ctx.Done():
					return
				}
			}
			if err != nil {
				select {
				case errch <- err:
					return
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	resultbuf := make([]byte, 0, 1<<17)
	for {
		select {
		case data := <-datach:
			// TODO(bassosimone): is there a more efficient way?
			resultbuf = append(resultbuf, data...)
		case err := <-errch:
			if errors.Is(err, io.EOF) {
				// see https://github.com/ooni/probe/issues/1965
				return resultbuf, nil
			}
			return resultbuf, netxlite.NewTopLevelGenericErrWrapper(err)
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.DeadlineExceeded) {
				return resultbuf, nil
			}
			return resultbuf, netxlite.NewTopLevelGenericErrWrapper(err)
		}
	}
}
