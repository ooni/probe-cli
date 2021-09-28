package netxlite

import (
	"context"
	"io"
)

// ReadAllContext is like io.ReadAll but reads r in a
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

// CopyContext is like io.Copy but may terminate earlier
// when the context expires. This function has the same
// caveats of ReadAllContext regarding the temporary leaking
// of the background goroutine used to do I/O.
func CopyContext(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
	countch, errch := make(chan int64, 1), make(chan error, 1) // buffers
	go func() {
		count, err := io.Copy(dst, src)
		if err != nil {
			errch <- err
			return
		}
		countch <- count
	}()
	select {
	case count := <-countch:
		return count, nil
	case <-ctx.Done():
		return 0, ctx.Err()
	case err := <-errch:
		return 0, err
	}
}
